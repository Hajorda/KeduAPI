package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const (
	projectID         = "keduapi"
	bucketName        = "bucket-quickstart_keduapi"
	maxFileSize       = 3 * 1024 * 1024 // 3 MB
	defaultUploadPath = "test-files/"
)

type ClientUploader struct {
	cl         *storage.Client
	projectID  string
	bucketName string
	uploadPath string
}

var (
	uploader       *ClientUploader
	rateLimiterMap = make(map[string]*rate.Limiter)
)

func init() {
	// Load sensitive credentials from environment variables
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "../../internal/keduapi-25ab2455b8d3.json")

	ctx := context.Background()

	client, err := storage.NewClient(ctx, option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	uploader = &ClientUploader{
		cl:         client,
		bucketName: bucketName,
		projectID:  projectID,
		uploadPath: defaultUploadPath,
	}
}

// Middleware for rate limiting by IP
func rateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if _, exists := rateLimiterMap[ip]; !exists {
			rateLimiterMap[ip] = rate.NewLimiter(3, 5) // Allow 1 request per second with a burst of 5
		}
		if !rateLimiterMap[ip].Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests, slow down"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// Middleware for logging requests
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Printf("[%s] %s %s %d %s",
			c.ClientIP(),
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			time.Since(start))
	}
}

func main() {
	r := gin.Default()
	r.Use(rateLimiter(), requestLogger())

	// Upload endpoint
	r.POST("/upload", func(c *gin.Context) {
		// Check file size
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxFileSize)
		f, err := c.FormFile("file_input")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File too large or missing"})
			return
		}

		// Validate file type
		if !strings.HasSuffix(strings.ToLower(f.Filename), ".jpg") && !strings.HasSuffix(strings.ToLower(f.Filename), ".jpeg") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Only JPG and JPEG files are allowed"})
			return
		}

		blobFile, err := f.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer blobFile.Close()

		err = uploader.UploadFile(blobFile, f.Filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Kedy uploaded successfully", "filename": f.Filename})
	})

	// Random image endpoint
	r.GET("/random-image", func(c *gin.Context) {
		image, metadata, err := uploader.GetRandomImage()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"image": image, "metadata": metadata})
	})

	// List images endpoint
	r.GET("/list-images", func(c *gin.Context) {
		images, err := uploader.ListImages()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"images": images})
	})

	// Deployment-ready server on port 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default to 8080 if no environment variable
	}
	r.Run(":" + port)
}

// UploadFile uploads an object
func (c *ClientUploader) UploadFile(file multipart.File, object string) error {
	ctx := context.Background()

	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	wc := c.cl.Bucket(c.bucketName).Object(c.uploadPath + object).NewWriter(ctx)
	if _, err := io.Copy(wc, file); err != nil {
		return fmt.Errorf("io.Copy: %v", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %v", err)
	}

	return nil
}

// GetRandomImage returns a random image and its metadata
func (c *ClientUploader) GetRandomImage() (string, map[string]interface{}, error) {
	ctx := context.Background()

	it := c.cl.Bucket(c.bucketName).Objects(ctx, &storage.Query{Prefix: c.uploadPath})
	var images []string
	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return "", nil, fmt.Errorf("iterator.Next: %v", err)
		}
		images = append(images, objAttrs.Name)
	}

	if len(images) == 0 {
		return "", nil, fmt.Errorf("no images found in bucket")
	}

	rand.Seed(time.Now().UnixNano())
	randomImage := images[rand.Intn(len(images))]

	obj := c.cl.Bucket(c.bucketName).Object(randomImage)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("object.Attrs: %v", err)
	}

	metadata := map[string]interface{}{
		"name":         attrs.Name,
		"content-type": attrs.ContentType,
		"size":         attrs.Size,
		"updated":      attrs.Updated,
	}

	return randomImage, metadata, nil
}

// ListImages lists all images in the bucket
func (c *ClientUploader) ListImages() ([]string, error) {
	ctx := context.Background()

	it := c.cl.Bucket(c.bucketName).Objects(ctx, &storage.Query{Prefix: c.uploadPath})
	var images []string
	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterator.Next: %v", err)
		}
		images = append(images, objAttrs.Name)
	}

	return images, nil
}
