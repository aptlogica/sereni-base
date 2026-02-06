package storage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	storageHttp "serenibase/internal/providers/storage/http"
	"serenibase/internal/providers/storage/interfaces"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewHTTPStorageClient tests the client constructor
func TestNewHTTPStorageClient(t *testing.T) {
	t.Run("create client with valid config", func(t *testing.T) {
		cfg := storageHttp.Config{
			BaseURL:        "http://localhost:8080",
			TimeoutSeconds: 30,
		}

		client, err := storageHttp.New(cfg)

		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("create client with empty base URL", func(t *testing.T) {
		cfg := storageHttp.Config{
			BaseURL:        "",
			TimeoutSeconds: 30,
		}

		client, err := storageHttp.New(cfg)

		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "base URL is required")
	})

	t.Run("create client with zero timeout uses default", func(t *testing.T) {
		cfg := storageHttp.Config{
			BaseURL:        "http://localhost:8080",
			TimeoutSeconds: 0,
		}

		client, err := storageHttp.New(cfg)

		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("create client with negative timeout uses default", func(t *testing.T) {
		cfg := storageHttp.Config{
			BaseURL:        "http://localhost:8080",
			TimeoutSeconds: -10,
		}

		client, err := storageHttp.New(cfg)

		require.NoError(t, err)
		assert.NotNil(t, client)
	})
}

// TestHTTPStorageClient_Upload tests file upload
func TestHTTPStorageClient_Upload(t *testing.T) {
	t.Run("successful upload", func(t *testing.T) {
		expectedResponse := interfaces.UploadResponse{
			Message: "File uploaded successfully",
			Path:    "uploads/test.txt",
			Url:     "http://storage/uploads/test.txt",
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/storage/upload", r.URL.Path)
			assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")

			// Parse multipart form
			err := r.ParseMultipartForm(10 << 20)
			require.NoError(t, err)

			// Check file exists
			file, header, err := r.FormFile("file")
			require.NoError(t, err)
			defer file.Close()

			assert.Equal(t, "test.txt", header.Filename)

			// Check path field
			path := r.FormValue("path")
			assert.Equal(t, "test.txt", path)

			// Read file content
			content, err := io.ReadAll(file)
			require.NoError(t, err)
			assert.Equal(t, "test file content", string(content))

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(expectedResponse)
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		reader := strings.NewReader("test file content")
		response, err := client.Upload(ctx, "test.txt", reader, int64(len("test file content")), "text/plain")

		require.NoError(t, err)
		assert.Equal(t, expectedResponse.Message, response.Message)
		assert.Equal(t, expectedResponse.Path, response.Path)
		assert.Equal(t, expectedResponse.Url, response.Url)
	})

	t.Run("upload with empty object name", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(10 << 20)
			require.NoError(t, err)

			_, header, err := r.FormFile("file")
			require.NoError(t, err)

			assert.Equal(t, "file", header.Filename)

			response := interfaces.UploadResponse{
				Message: "success",
				Path:    "file",
				Url:     "http://storage/file",
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		reader := strings.NewReader("content")
		response, err := client.Upload(ctx, "file", reader, 7, "text/plain")

		require.NoError(t, err)
		assert.Equal(t, "success", response.Message)
		assert.Equal(t, "file", response.Path)
		assert.Equal(t, "http://storage/file", response.Url)
	})

	t.Run("upload with server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"internal server error"}`))
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		reader := strings.NewReader("test content")
		_, err = client.Upload(ctx, "test.txt", reader, 12, "text/plain")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "upload failed with status 500")
	})

	t.Run("upload with network error", func(t *testing.T) {
		cfg := storageHttp.Config{
			BaseURL:        "http://invalid-host-does-not-exist:9999",
			TimeoutSeconds: 1,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		reader := strings.NewReader("test content")
		_, err = client.Upload(ctx, "test.txt", reader, 12, "text/plain")

		assert.Error(t, err)
	})

	t.Run("upload with invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`invalid json`))
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		reader := strings.NewReader("test content")
		_, err = client.Upload(ctx, "test.txt", reader, 12, "text/plain")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode response")
	})

	t.Run("upload large file", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(10 << 20)
			require.NoError(t, err)

			file, _, err := r.FormFile("file")
			require.NoError(t, err)
			defer file.Close()

			content, err := io.ReadAll(file)
			require.NoError(t, err)
			assert.Equal(t, 1024, len(content))

			response := interfaces.UploadResponse{
				Message: "success",
				Path:    "large.bin",
				Url:     "http://storage/large.bin",
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		largeContent := bytes.Repeat([]byte("x"), 1024)
		reader := bytes.NewReader(largeContent)
		_, err = client.Upload(ctx, "large.bin", reader, int64(len(largeContent)), "application/octet-stream")

		require.NoError(t, err)
	})

	t.Run("upload with context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Should not reach here
			t.Fatal("Request should be cancelled")
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		reader := strings.NewReader("test content")
		_, err = client.Upload(ctx, "test.txt", reader, 12, "text/plain")

		assert.Error(t, err)
	})
}

// TestHTTPStorageClient_Download tests file download
func TestHTTPStorageClient_Download(t *testing.T) {
	t.Run("successful download", func(t *testing.T) {
		expectedContent := "downloaded file content"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/storage/download", r.URL.Path)
			assert.Equal(t, "test.txt", r.URL.Query().Get("path"))

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(expectedContent))
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		readCloser, err := client.Download(ctx, "test.txt")

		require.NoError(t, err)
		require.NotNil(t, readCloser)
		defer readCloser.Close()

		content, err := io.ReadAll(readCloser)
		require.NoError(t, err)
		assert.Equal(t, expectedContent, string(content))
	})

	t.Run("download with special characters in path", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Query().Get("path")
			assert.Equal(t, "folder/file with spaces.txt", path)

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("content"))
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		readCloser, err := client.Download(ctx, "folder/file with spaces.txt")

		require.NoError(t, err)
		require.NotNil(t, readCloser)
		defer readCloser.Close()
	})

	t.Run("download non-existent file", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"file not found"}`))
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		_, err = client.Download(ctx, "nonexistent.txt")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "download failed with status 404")
	})

	t.Run("download with server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"server error"}`))
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		_, err = client.Download(ctx, "test.txt")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})

	t.Run("download with network error", func(t *testing.T) {
		cfg := storageHttp.Config{
			BaseURL:        "http://invalid-host:9999",
			TimeoutSeconds: 1,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		_, err = client.Download(ctx, "test.txt")

		assert.Error(t, err)
	})

	t.Run("download with context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("Request should be cancelled")
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err = client.Download(ctx, "test.txt")

		assert.Error(t, err)
	})
}

// TestHTTPStorageClient_Delete tests file deletion
func TestHTTPStorageClient_Delete(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodDelete, r.Method)
			assert.Equal(t, "/storage/delete", r.URL.Path)
			assert.Equal(t, "test.txt", r.URL.Query().Get("path"))

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message":"deleted"}`))
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		err = client.Delete(ctx, "test.txt")

		require.NoError(t, err)
	})

	t.Run("delete with special characters", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Query().Get("path")
			assert.Equal(t, "folder/file@#$.txt", path)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		err = client.Delete(ctx, "folder/file@#$.txt")

		require.NoError(t, err)
	})

	t.Run("delete non-existent file", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"file not found"}`))
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		err = client.Delete(ctx, "nonexistent.txt")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "delete failed with status 404")
	})

	t.Run("delete with server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"server error"}`))
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		err = client.Delete(ctx, "test.txt")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})

	t.Run("delete with network error", func(t *testing.T) {
		cfg := storageHttp.Config{
			BaseURL:        "http://invalid-host:9999",
			TimeoutSeconds: 1,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		err = client.Delete(ctx, "test.txt")

		assert.Error(t, err)
	})

	t.Run("delete with context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("Request should be cancelled")
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = client.Delete(ctx, "test.txt")

		assert.Error(t, err)
	})
}

// TestHTTPStorageClient_Exists tests file existence check
func TestHTTPStorageClient_Exists(t *testing.T) {
	t.Run("file exists", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/storage/download", r.URL.Path)

			w.WriteHeader(http.StatusOK)
			w.Write([]byte("file content"))
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		exists, err := client.Exists(ctx, "test.txt")

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("file does not exist", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found"}`))
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		exists, err := client.Exists(ctx, "nonexistent.txt")

		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("exists check with server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"server error"}`))
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		exists, err := client.Exists(ctx, "test.txt")

		assert.Error(t, err)
		assert.False(t, exists)
		assert.Contains(t, err.Error(), "exists check failed with status 500")
	})

	t.Run("exists check with network error", func(t *testing.T) {
		cfg := storageHttp.Config{
			BaseURL:        "http://invalid-host:9999",
			TimeoutSeconds: 1,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		exists, err := client.Exists(ctx, "test.txt")

		assert.Error(t, err)
		assert.False(t, exists)
	})

	t.Run("exists check with context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("Request should be cancelled")
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		exists, err := client.Exists(ctx, "test.txt")

		assert.Error(t, err)
		assert.False(t, exists)
	})
}

// TestHTTPStorageClient_EdgeCases tests edge cases
func TestHTTPStorageClient_EdgeCases(t *testing.T) {
	t.Run("empty object name", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Query().Get("path")
			assert.Equal(t, "", path)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		exists, err := client.Exists(ctx, "")

		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("very long object name", func(t *testing.T) {
		longName := strings.Repeat("a", 1000) + ".txt"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Query().Get("path")
			assert.Equal(t, longName, path)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("content"))
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		readCloser, err := client.Download(ctx, longName)

		require.NoError(t, err)
		require.NotNil(t, readCloser)
		readCloser.Close()
	})

	t.Run("unicode in object name", func(t *testing.T) {
		unicodeName := "测试文件.txt"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Query().Get("path")
			assert.Equal(t, unicodeName, path)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		cfg := storageHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := storageHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		err = client.Delete(ctx, unicodeName)

		require.NoError(t, err)
	})
}
