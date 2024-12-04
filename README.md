# KeduAPI Documentation

KeduAPI is a simple API for uploading and retrieving images stored in Google Cloud Storage.

## Base URL

```
https://keduapi-1002036435905.europe-west3.run.app
```

## Endpoints

### Upload an Image

**URL:** `/upload`  
**Method:** `POST`  
**Description:** Uploads a JPEG image to the server.

**Form Data:**

- `file_input`: The image file to upload. Must be a `.jpg` or `.jpeg` file and not exceed 3 MB in size.

**Response:**

- `200 OK`  
  ```json
  {
    "message": "File uploaded successfully"
  }
  ```

**Error Responses:**

- `400 Bad Request` if the file is missing, exceeds size limit, or is of an unsupported type.
- `500 Internal Server Error` for server-side issues.

---

### List All Images

**URL:** `/list-images`  
**Method:** `GET`  
**Description:** Retrieves a list of all image names stored on the server.

**Response:**

- `200 OK`  
  ```json
  {
    "images": [
      "image1.jpg",
      "image2.jpg",
      "...",
      "imageN.jpg"
    ]
  }
  ```

**Error Responses:**

- `500 Internal Server Error` if images cannot be retrieved.

---

### Get a Specific Image

**URL:** `/get-image/:name`  
**Method:** `GET`  
**Description:** Retrieves the URL and metadata of a specific image.

**URL Parameters:**

- `:name` (string): The name of the image file.

**Response:**

- `200 OK`  
  ```json
  {
    "image_url": "https://storage.googleapis.com/your-bucket-name/path/to/image.jpg",
    "metadata": {
      "ContentType": "image/jpeg",
      "Size": 123456,
      "Updated": "2023-12-04T12:00:00Z",
      "OtherMetadata": "..."
    }
  }
  ```

**Error Responses:**

- `404 Not Found` if the image does not exist.
- `500 Internal Server Error` for server-side issues.

---

### Get a Random Image

**URL:** `/random-image`  
**Method:** `GET`  
**Description:** Retrieves a random image's URL and metadata from the stored images.

**Response:**

- `200 OK`  
  ```json
  {
    "image_url": "https://storage.googleapis.com/your-bucket-name/path/to/random-image.jpg",
    "metadata": {
      "ContentType": "image/jpeg",
      "Size": 789012,
      "Updated": "2023-12-04T12:34:56Z",
      "OtherMetadata": "..."
    }
  }
  ```

**Error Responses:**

- `500 Internal Server Error` if no images are found or upon failure.

---

### Health Check

**URL:** `/health`  
**Method:** `GET`  
**Description:** Checks the health status of the API and its dependencies.

**Response:**

- `200 OK`  
  ```json
  {
    "status": "healthy",
    "details": {
      "redis": "healthy",
      "google-cloud-storage": "healthy"
    }
  }
  ```

**Error Responses:**

- `200 OK` with status `"unhealthy"` if any dependencies are not functioning properly.

---

## Error Handling

The API returns errors in JSON format with appropriate HTTP status codes.

**Example Error Response:**

- `400 Bad Request`  
  ```json
  {
    "error": "Description of the error"
  }
  ```

---

## Notes

- **File Restrictions:** Only `.jpg` and `.jpeg` files are accepted, with a maximum size of 3 MB.
- **Caching:** Image lists are cached in Redis for improved performance.
- **Authentication:** The API is currently configured to allow unauthenticated access.
- **Environment:** Ensure that the Google Cloud credentials and Redis configurations are properly set in the production environment.

---
