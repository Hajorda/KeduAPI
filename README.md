# KeduAPI Documentation 🐱✨

Welcome to the **KeduAPI** documentation! This is a simple yet powerful API designed to help you upload, manage, and retrieve kedy images from our server. Whether you want to upload your favorite kedy photos or retrieve random kedys for some extra flair, this API has you covered. 🚀🐾

Let's dive into the details!

## Base URL

```
https://api.keduapp.online
```

---

## Endpoints

### 1. Upload an Image 📤

**URL:** `/upload`  
**Method:** `POST`  
**Description:** Uploads a JPEG image (or your cat's best photo) to the server. The file must be `.jpg` or `.jpeg` and shouldn't exceed 3 MB.

#### Request Details

- **Content-Type**: `multipart/form-data`
- **Form Data**:
  - `file_input`: The image file to upload (only `.jpg` or `.jpeg` supported).

#### Example Request (using `curl`)

```bash
curl -X POST "https://api.keduapp.online/upload" \
  -F "file_input=@/path/to/your/image.jpg"
```

#### Example Request (using `React` and `axios`)

```javascript
import axios from 'axios';

const uploadImage = async (file) => {
  const formData = new FormData();
  formData.append('file_input', file);

  try {
    const response = await axios.post('https://api.keduapp.online/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data'
      }
    });
    console.log('Upload success:', response.data);
  } catch (error) {
    console.error('Upload failed:', error.response ? error.response.data : error.message);
  }
};
```

#### Response:

- `200 OK`  
  ```json
  {
    "message": "File uploaded successfully"
  }
  ```

#### Error Responses:

- `400 Bad Request` if the file is missing, exceeds size limit, or is an unsupported type.
- `500 Internal Server Error` for server-side issues.

---

### 2. List All Images 🖼️

**URL:** `/list-images`  
**Method:** `GET`  
**Description:** Retrieves a list of all image names stored on the server.

#### Example Request (using `curl`)

```bash
curl -X GET "https://api.keduapp.online/list-images"
```

#### Example Request (using `React` and `axios`)

```javascript
const listImages = async () => {
  try {
    const response = await axios.get('https://api.keduapp.online/list-images');
    console.log('Images:', response.data.images);
  } catch (error) {
    console.error('Error listing images:', error.response ? error.response.data : error.message);
  }
};
```

#### Response:

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

#### Error Responses:

- `500 Internal Server Error` if images cannot be retrieved.

---

### 3. Get a Specific Image 📸

**URL:** `/get-image/:name`  
**Method:** `GET`  
**Description:** Retrieves the URL and metadata of a specific image by its name.

#### URL Parameters:

- `:name` (string): The name of the image file.

#### Example Request (using `curl`)

```bash
curl -X GET "https://api.keduapp.online/get-image/image1.jpg"
```

#### Example Request (using `React` and `axios`)

```javascript
const getImage = async (imageName) => {
  try {
    const response = await axios.get(`https://api.keduapp.online/get-image/${imageName}`);
    console.log('Image URL:', response.data.image_url);
    console.log('Metadata:', response.data.metadata);
  } catch (error) {
    console.error('Error fetching image:', error.response ? error.response.data : error.message);
  }
};
```

#### Response:

- `200 OK`  
  ```json
  {
    "image_url": "https://storage.googleapis.com/your-bucket-name/path/to/image.jpg",
    "metadata": {
      "name": "image1.jpg",
      "content-type": "image/jpeg",
      "size": 123456,
      "updated": "2023-12-04T12:00:00Z"
    }
  }
  ```

#### Error Responses:

- `404 Not Found` if the image does not exist.
- `500 Internal Server Error` for server-side issues.

---

### 4. Get a Random Image 🌲

**URL:** `/random-image`  
**Method:** `GET`  
**Description:** Retrieves a random image's URL and metadata from the stored images. Who knows, you might get a random cat gif! 🐱💨

#### Example Request (using `curl`)

```bash
curl -X GET "https://api.keduapp.online/random-image"
```

#### Example Request (using `React` and `axios`)

```javascript
const getRandomImage = async () => {
  try {
    const response = await axios.get('https://api.keduapp.online/random-image');
    console.log('Random Image URL:', response.data.image_url);
    console.log('Metadata:', response.data.metadata);
  } catch (error) {
    console.error('Error fetching random image:', error.response ? error.response.data : error.message);
  }
};
```

#### Response:

- `200 OK`  
  ```json
  {
    "image_url": "https://storage.googleapis.com/your-bucket-name/path/to/random-image.jpg",
    "metadata": {
      "name": "random-image.jpg",
      "content-type": "image/jpeg",
      "size": 789012,
      "updated": "2023-12-04T12:34:56Z"
    }
  }
  ```

#### Error Responses:

- `500 Internal Server Error` if no images are found or upon failure.

---

### 5. Health Check 🏥

**URL:** `/health`  
**Method:** `GET`  
**Description:** Checks the health status of the API and its dependencies (like Google Cloud Storage).

#### Example Request (using `curl`)

```bash
curl -X GET "https://api.keduapp.online/health"
```

#### Example Request (using `React` and `axios`)

```javascript
const checkHealth = async () => {
  try {
    const response = await axios.get('https://api.keduapp.online/health');
    console.log('Health Status:', response.data);
  } catch (error) {
    console.error('Error checking health:', error.response ? error.response.data : error.message);
  }
};
```

#### Response:

- `200 OK`  
  ```json
  {
    "status": "healthy",
    "details": {
      "google-cloud-storage": "healthy"
    }
  }
  ```

---

## Error Handling ⚠️

The API returns errors in JSON format with appropriate HTTP status codes.

**Example Error Response:**

- `400 Bad Request`  
  ```json
  {
    "error": "Description of the error"
  }
  ```

---

## Notes 📁

- **File Restrictions:** Only `.jpg` and `.jpeg` files are accepted, with a maximum size of 3 MB.
- **Caching:** Image lists are cached in Redis for improved performance.
- **Authentication:** The API is currently configured to allow unauthenticated access.
- **Environment:** Ensure that the Google Cloud credentials are properly set in the production environment.

---

### Here's your bonus fun cat gif! 🐱🎉

![Turning Cat GIF](https://media.tenor.com/T8SBWIfKcFQAAAAj/maxwell-cat.gif)

Hope this documentation brought a smile to your face! 😄

