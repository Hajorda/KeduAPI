package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	_ "image/jpeg"
	_ "image/png"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	firebase "firebase.google.com/go"
	"github.com/corona10/goimagehash"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/exp/rand"
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
	cl          *storage.Client
	projectID   string
	bucketName  string
	uploadPath  string
	firestoreDB *firestore.Client
}

// Define the Image struct for Firestore
type Image struct {
	ImageHash string `firestore:"imageHash"`
	ImageName string `firestore:"imageName"`
}

var uploader *ClientUploader

const (
	firebaseConfigFile = "keduapi-15ac7bd4be11.json"
	firebaseDBURL      = "https://keduapi-cb376-default-rtdb.europe-west1.firebasedatabase.app/"
)

func init() {
	ctx := context.Background()

	// Initialize Google Cloud Storage client
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Printf("Failed to create Google Cloud Storage client: %v", err)
	}

	// Initialize Firebase
	ctx = context.Background()
	opt := option.WithCredentialsFile(firebaseConfigFile)
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID}, opt)

	if err != nil {
		log.Fatalf("Firebase initialization error: %v\n", err)
	}

	// Initialize Firestore
	firestoreClient, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalf("Firestore initialization error: %v\n", err)
	}

	uploader = &ClientUploader{
		cl:          client,
		projectID:   projectID,
		bucketName:  bucketName,
		uploadPath:  defaultUploadPath,
		firestoreDB: firestoreClient,
	}
}

func main() {

	r := gin.Default()

	// Enable CORS
	r.Use(cors.Default())

	// Serve static files
	r.Static("/public", "./public")

	// Serve the documentation at the root URL
	r.GET("/", func(c *gin.Context) {
		c.File("./public/index.html")
	})

	r.POST("/verify-captcha", VerifyCaptcha)

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

		rand.Seed(uint64(time.Now().UnixNano()))
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

	// ozhanDetector checks if the user's IP is associated with Bilkent University ASN (AS8466)
	r.GET("/ozhan-detector", func(c *gin.Context) {
		userIP := c.ClientIP() // Get user's IP address
		// print the user's IP address for debugging
		log.Printf("User's IP address: %s", userIP)
		// Check if the user is associated with Bilkent University ASN
		isOzhanDetected, err := isItOzhan(userIP)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify ASN", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"isItOzhan": isOzhanDetected,
		})
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

// isItOzhan checks if the user's IP is associated with Bilkent University ASN (AS8466)
func isItOzhan(userIP string) (bool, error) {
	// Call an external IP geolocation API (e.g., IPinfo) to get the ASN for the user's IP address
	apiURL := fmt.Sprintf("https://ipinfo.io/%s/json?token=8b3199dacefb30", userIP)
	resp, err := http.Get(apiURL)
	if err != nil {
		return false, fmt.Errorf("failed to get ASN data: %v", err)
	}
	defer resp.Body.Close()

	var data struct {
		ASN struct {
			ASN string `json:"asn"`
		} `json:"asn"`
	}

	// Decode the response body
	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &data)
	if err != nil {
		return false, fmt.Errorf("failed to parse ASN data: %v", err)
	}

	// Check if the ASN is AS8466 (Bilkent University)
	if data.ASN.ASN == "AS8466" {
		return true, nil
	}

	return false, nil
}

// VerifyCaptcha verifies the captcha token using Cloudflare Turnstile
func VerifyCaptcha(c *gin.Context) {
	var reqBody struct {
		Token string `json:"token"`
	}

	// Parse incoming JSON request
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}

	// Validate the token with Cloudflare Turnstile API
	secretKey := "0x4AAAAAAA4niU1crrMPhzvp4comG9pcJgs"
	if secretKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Secret key not set"})
		return
	}
	url := "https://challenges.cloudflare.com/turnstile/v0/siteverify"

	// Get the client's IP address
	ip := c.ClientIP()

	// Make POST request to verify captcha
	resp, err := http.PostForm(url, map[string][]string{
		"secret":   {secretKey},
		"response": {reqBody.Token},
		"remoteip": {ip},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to verify captcha"})
		return
	}
	defer resp.Body.Close()

	// Read and print the full response body for debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to read captcha response"})
		return
	}

	// Read and parse the response
	var captchaResponse struct {
		Success    bool     `json:"success"`
		ErrorCodes []string `json:"error-codes"`
	}
	if err := json.Unmarshal(body, &captchaResponse); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to parse captcha response"})
		return
	}

	// Check if captcha validation was successful
	if !captchaResponse.Success {
		c.JSON(http.StatusForbidden, gin.H{
			"message":     "Captcha validation failed",
			"error_codes": captchaResponse.ErrorCodes,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Captcha validated successfully"})
}

func (c *ClientUploader) computeHash(img multipart.File) (*goimagehash.ImageHash, error) {
	// Read the file into memory
	data, err := io.ReadAll(img)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Decode the image
	imgReader := bytes.NewReader(data)
	imgDecoded, _, err := image.Decode(imgReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %v", err)
	}

	// Compute perceptual hash
	hash, err := goimagehash.PerceptionHash(imgDecoded)
	if err != nil {
		return nil, fmt.Errorf("failed to compute image hash: %v", err)
	}

	return hash, nil
}

// UploadFile uploads a file to Google Cloud Storage
func (c *ClientUploader) UploadFile(file multipart.File, object string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*50)
	defer cancel()

	// Compute hash
	file.Seek(0, io.SeekStart) // Reset file pointer
	hash, err := c.computeHash(file)
	if err != nil {
		return fmt.Errorf("failed to compute image hash: %v", err)
	}

	// Check for duplicates in Firestore
	imageHash := hash.GetHash()
	imageHashStr := strings.TrimSpace(fmt.Sprintf("%d", imageHash))

	fmt.Println("Debug: Checking for duplicate image with hash:", imageHashStr)

	query := c.firestoreDB.Collection("images").Where("imageHash", "==", imageHashStr).Limit(1)
	iter := query.Documents(ctx)
	defer iter.Stop()

	// // Print query reference for debugging
	// fmt.Println("Debug: Firestore Query Reference:", query)

	doc, err := iter.Next()

	if err == nil {
		fmt.Println("Debug: Duplicate image found in Firestore. Document ID:", doc.Ref.ID)

		// Print the stored hash from Firestore
		storedHash, err := doc.DataAt("image_hash")
		if err == nil {
			fmt.Println("Debug: Stored hash in Firestore:", storedHash)
		} else {
			fmt.Println("Debug: Failed to read stored hash:", err)
		}

		return fmt.Errorf("duplicate image detected")
	} else if err == iterator.Done {
		fmt.Println("Debug: No duplicate found, proceeding with upload.")
	} else {
		fmt.Println("Error querying Firestore:", err)
		return fmt.Errorf("error querying Firestore: %v", err)
	}

	// No duplicate found, continue

	// // Fetch all images to inspect stored values
	// iter = c.firestoreDB.Collection("images").Documents(ctx)
	// defer iter.Stop()

	// fmt.Println("Debug: Fetching all stored images to check stored hashes.")
	// for {
	// 	doc, err := iter.Next()
	// 	if err == iterator.Done {
	// 		break
	// 	}
	// 	if err != nil {
	// 		fmt.Println("Error fetching images:", err)
	// 		return fmt.Errorf("error fetching images: %v", err)
	// 	}

	// 	storedHash, _ := doc.DataAt("image_hash")
	// 	fmt.Printf("Debug: Stored hash in Firestore: [%v] (type: %T), Document ID: %s\n", storedHash, storedHash, doc.Ref.ID)
	// }

	// Upload the image to Cloud Storage
	file.Seek(0, io.SeekStart) // Reset file pointer again
	wc := c.cl.Bucket(c.bucketName).Object(c.uploadPath + object).NewWriter(ctx)
	if _, err := io.Copy(wc, file); err != nil {
		return fmt.Errorf("io.Copy: %v", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("Writer.Close: %v", err)
	}

	// Store the hash and image name in Firestore
	_, err = c.firestoreDB.Collection("images").Doc(object).Set(ctx, Image{ImageHash: fmt.Sprintf("%d", hash.GetHash()), ImageName: object})
	if err != nil {
		return fmt.Errorf("failed to store image in Firestore: %v", err)
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
