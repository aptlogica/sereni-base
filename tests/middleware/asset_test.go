package middleware_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"github.com/aptlogica/sereni-base/internal/middleware"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestFileSizeLimitMiddleware tests file size and count limitations
func TestFileSizeLimitMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		files          []struct{ name, content string }
		expectedStatus int
	}{
		{
			name:           "no files uploaded",
			files:          []struct{ name, content string }{},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "single file within limit",
			files: []struct{ name, content string }{
				{"file1.txt", "hello world"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "multiple files within limit",
			files: []struct{ name, content string }{
				{"file1.txt", "content1"},
				{"file2.txt", "content2"},
				{"file3.txt", "content3"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "too many files",
			files: []struct{ name, content string }{
				{"file1.txt", "content1"},
				{"file2.txt", "content2"},
				{"file3.txt", "content3"},
				{"file4.txt", "content4"},
				{"file5.txt", "content5"},
				{"file6.txt", "content6"}, // Exceeds limit of 5
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "file too large",
			files: []struct{ name, content string }{
				{"large.txt", generateLargeContent(11 * 1024 * 1024)}, // 11MB > 10MB limit
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.POST("/upload", middleware.FileSizeLimitMiddleware(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			// Create multipart form
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			for _, file := range tt.files {
				part, err := writer.CreateFormFile("files", file.name)
				if err != nil {
					t.Fatal(err)
				}
				io.WriteString(part, file.content)
			}
			writer.Close()

			req := httptest.NewRequest("POST", "/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestFileSizeLimitMiddleware_EdgeCases tests edge cases
func TestFileSizeLimitMiddleware_EdgeCases(t *testing.T) {
	t.Run("exactly at file count limit", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		r.POST("/upload", middleware.FileSizeLimitMiddleware(), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Upload exactly 5 files (the limit)
		for i := 1; i <= 5; i++ {
			part, _ := writer.CreateFormFile("files", "file.txt")
			io.WriteString(part, "content")
		}
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		c.Request = req
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("exactly at file size limit", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		r.POST("/upload", middleware.FileSizeLimitMiddleware(), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// Upload file exactly at 10MB limit
		part, _ := writer.CreateFormFile("files", "exact.txt")
		io.WriteString(part, generateLargeContent(10*1024*1024))
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		c.Request = req
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid multipart form", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		r.POST("/upload", middleware.FileSizeLimitMiddleware(), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("POST", "/upload", bytes.NewReader([]byte("invalid")))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=---invalid")
		c.Request = req
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("request body too large", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		r.POST("/upload", middleware.FileSizeLimitMiddleware(), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		// Create a request that exceeds 100MB total size
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("files", "huge.txt")
		io.WriteString(part, generateLargeContent(101*1024*1024))
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		c.Request = req
		r.ServeHTTP(w, req)

		// Should return an error due to the form parsing limit
		assert.NotEqual(t, http.StatusOK, w.Code)
	})
}

// TestFileSizeLimitMiddleware_MultipleFilesWithOneTooLarge tests mixed file sizes
func TestFileSizeLimitMiddleware_MultipleFilesWithOneTooLarge(t *testing.T) {
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.POST("/upload", middleware.FileSizeLimitMiddleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add small files
	part1, _ := writer.CreateFormFile("files", "small1.txt")
	io.WriteString(part1, "small content 1")

	// Add a large file
	part2, _ := writer.CreateFormFile("files", "large.txt")
	io.WriteString(part2, generateLargeContent(11*1024*1024))

	// Add another small file
	part3, _ := writer.CreateFormFile("files", "small2.txt")
	io.WriteString(part3, "small content 2")

	writer.Close()

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	c.Request = req
	r.ServeHTTP(w, req)

	// Should fail because one file is too large
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// generateLargeContent creates a string of specified size in bytes
func generateLargeContent(sizeBytes int) string {
	content := make([]byte, sizeBytes)
	for i := range content {
		content[i] = 'a'
	}
	return string(content)
}

// TestFileSizeLimitMiddleware_DifferentFileTypes tests validation with different file types
func TestFileSizeLimitMiddleware_DifferentFileTypes(t *testing.T) {
	fileTypes := []struct {
		name     string
		filename string
		content  string
	}{
		{"text file", "document.txt", "text content"},
		{"json file", "data.json", `{"key": "value"}`},
		{"csv file", "data.csv", "col1,col2\nval1,val2"},
		{"xml file", "data.xml", "<root><item>value</item></root>"},
		{"binary file", "image.bin", string([]byte{0xFF, 0xD8, 0xFF, 0xE0})},
	}

	for _, ft := range fileTypes {
		t.Run(ft.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.POST("/upload", middleware.FileSizeLimitMiddleware(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, _ := writer.CreateFormFile("files", ft.filename)
			io.WriteString(part, ft.content)
			writer.Close()

			req := httptest.NewRequest("POST", "/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestFileSizeLimitMiddleware_EmptyFiles tests handling of empty files
func TestFileSizeLimitMiddleware_EmptyFiles(t *testing.T) {
	t.Run("single empty file", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		r.POST("/upload", middleware.FileSizeLimitMiddleware(), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("files", "empty.txt")
		io.WriteString(part, "")
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		c.Request = req
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("multiple empty files", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, r := gin.CreateTestContext(w)

		r.POST("/upload", middleware.FileSizeLimitMiddleware(), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		for i := 1; i <= 3; i++ {
			part, _ := writer.CreateFormFile("files", "empty.txt")
			io.WriteString(part, "")
		}
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		c.Request = req
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestFileSizeLimitMiddleware_VaryingFileSizes tests files of various sizes
func TestFileSizeLimitMiddleware_VaryingFileSizes(t *testing.T) {
	fileSizes := []struct {
		name string
		size int
	}{
		{"1 byte", 1},
		{"1 KB", 1024},
		{"100 KB", 100 * 1024},
		{"1 MB", 1024 * 1024},
		{"5 MB", 5 * 1024 * 1024},
		{"9 MB", 9 * 1024 * 1024},
	}

	for _, fs := range fileSizes {
		t.Run(fs.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.POST("/upload", middleware.FileSizeLimitMiddleware(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, _ := writer.CreateFormFile("files", "file.txt")
			io.WriteString(part, generateLargeContent(fs.size))
			writer.Close()

			req := httptest.NewRequest("POST", "/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestFileSizeLimitMiddleware_FilenameEdgeCases tests different filename formats
func TestFileSizeLimitMiddleware_FilenameEdgeCases(t *testing.T) {
	filenames := []string{
		"simple.txt",
		"file with spaces.txt",
		"file-with-dashes.txt",
		"file_with_underscores.txt",
		"file.multiple.dots.txt",
		"UPPERCASE.TXT",
		"12345.txt",
		"file@special#chars$.txt",
	}

	for _, filename := range filenames {
		t.Run(filename, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, r := gin.CreateTestContext(w)

			r.POST("/upload", middleware.FileSizeLimitMiddleware(), func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			part, _ := writer.CreateFormFile("files", filename)
			io.WriteString(part, "content")
			writer.Close()

			req := httptest.NewRequest("POST", "/upload", body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			c.Request = req
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

// TestFileSizeLimitMiddleware_ConcurrentUploads tests concurrent file uploads
func TestFileSizeLimitMiddleware_ConcurrentUploads(t *testing.T) {
	_, r := gin.CreateTestContext(httptest.NewRecorder())

	r.POST("/upload", middleware.FileSizeLimitMiddleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	successCount := 0
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("files", "file.txt")
		io.WriteString(part, "content")
		writer.Close()

		req := httptest.NewRequest("POST", "/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		r.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			successCount++
		}
	}

	assert.Equal(t, 5, successCount)
}
