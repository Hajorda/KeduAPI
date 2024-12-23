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
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const (
	projectID         = "keduapi"
	bucketName        = "bucket-quickstart_keduapi"
	maxFileSize       = 3 * 1024 * 1024 // 3 MB
	defaultUploadPath = "kedys/"
)

type ClientUploader struct {
	cl         *storage.Client
	projectID  string
	bucketName string
	uploadPath string
}

var uploader *ClientUploader

func init() {
	ctx := context.Background()

	// Initialize Google Cloud Storage client
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Printf("Failed to create Google Cloud Storage client: %v", err)
	}

	uploader = &ClientUploader{
		cl:         client,
		projectID:  projectID,
		bucketName: bucketName,
		uploadPath: defaultUploadPath,
	}
}

func main() {
	r := gin.Default()

	// Upload an image
	r.POST("/upload", func(c *gin.Context) {
		f, err := c.FormFile("file_input")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if f.Size > maxFileSize {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File size exceeds the 3MB limit"})
			return
		}

		if !isAllowedFileType(f.Filename) {
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

		c.JSON(http.StatusOK, gin.H{"message": "Kedy uploaded successfully"})
	})

	// List all images
	r.GET("/list-images", func(c *gin.Context) {
		images, err := uploader.ListImages()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"images": images})
	})

	// Get a specific image by name
	r.GET("/get-image/:name", func(c *gin.Context) {
		imageName := c.Param("name")

		imageURL, metadata, err := uploader.GetSpecificImage(imageName)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Image not found", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"image_url": imageURL, "metadata": metadata})
	})

	// Get a random image
	r.GET("/random-image", func(c *gin.Context) {
		images, err := uploader.ListImages()
		if err != nil || len(images) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No images found or failed to retrieve images"})
			return
		}
// guzelş
		rand.Seed(time.Now().UnixNano())
		randomIndex := rand.Intn(len(images))
		randomImage := images[randomIndex]

		log.Printf("Random image selected: %s", randomImage)

		imageURL, metadata, err := uploader.GetSpecificImage(randomImage)
		if err != nil {
			log.Printf("Error retrieving image metadata: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve image metadata", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"image_url": imageURL, "metadata": metadata})
	})

	// Health check endpoint
	r.GET("/health", uploader.HealthCheck())

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port if not specified
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

// UploadFile uploads a file to Google Cloud Storage
func (c *ClientUploader) UploadFile(file multipart.File, object string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*50)
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

// GetSpecificImage retrieves a specific image's metadata
func (c *ClientUploader) GetSpecificImage(imageName string) (string, map[string]interface{}, error) {
	ctx := context.Background()
	objectName := strings.TrimPrefix(imageName, c.uploadPath)

	obj := c.cl.Bucket(c.bucketName).Object(c.uploadPath + objectName)
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

	publicURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", c.bucketName, attrs.Name)

	return publicURL, metadata, nil
}

// HealthCheck checks the health of Google Cloud Storage
func (c *ClientUploader) HealthCheck() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		health := map[string]string{}

		// Check Google Cloud Storage
		if _, err := c.cl.Bucket(c.bucketName).Attrs(context.Background()); err != nil {
			health["google-cloud-storage"] = "unhealthy"
		} else {
			health["google-cloud-storage"] = "healthy"
		}

		ctx.JSON(http.StatusOK, gin.H{"status": "healthy", "details": health})
	}
}

// Helper function to check allowed file types
func isAllowedFileType(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.HasSuffix(lower, ".jpg") || strings.HasSuffix(lower, ".jpeg")
}

func makeObjectPublic(bucketName, objectName string) error {
	ctx := context.Background()
	client, err := storage.NewClient(ctx, option.WithCredentialsFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")))
	if err != nil {
		return err
	}
	defer client.Close()

	bucket := client.Bucket(bucketName)
	obj := bucket.Object(objectName)

	// Make the object publicly readable
	if err := obj.ACL().Set(ctx, storage.AllUsers, storage.RoleReader); err != nil {
		return err
	}

	return nil
}
