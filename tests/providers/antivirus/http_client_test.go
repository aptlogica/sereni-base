package antivirus_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	antivirusHttp "serenibase/internal/providers/antivirus/http"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testBaseURL    = "http://localhost:8080"
	testCleanFile  = "clean.txt"
	testVirusFile  = "virus.exe"
	testThreatName = "Trojan.Generic.123456"
	testLargeFile  = "large.bin"
	testFileName   = "test.txt"
)

// TestNewHTTPAntivirusClient tests the client constructor
func TestNewHTTPAntivirusClient(t *testing.T) {
	t.Run("create client with valid config", func(t *testing.T) {
		cfg := antivirusHttp.Config{
			BaseURL:        testBaseURL,
			TimeoutSeconds: 30,
		}

		client, err := antivirusHttp.New(cfg)

		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("create client with empty base URL", func(t *testing.T) {
		cfg := antivirusHttp.Config{
			BaseURL:        "",
			TimeoutSeconds: 30,
		}

		client, err := antivirusHttp.New(cfg)

		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "base URL is required")
	})

	t.Run("create client with zero timeout uses default", func(t *testing.T) {
		cfg := antivirusHttp.Config{
			BaseURL:        testBaseURL,
			TimeoutSeconds: 0,
		}

		client, err := antivirusHttp.New(cfg)

		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("create client with negative timeout uses default", func(t *testing.T) {
		cfg := antivirusHttp.Config{
			BaseURL:        testBaseURL,
			TimeoutSeconds: -10,
		}

		client, err := antivirusHttp.New(cfg)

		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("create client with custom timeout", func(t *testing.T) {
		cfg := antivirusHttp.Config{
			BaseURL:        testBaseURL,
			TimeoutSeconds: 60,
		}

		client, err := antivirusHttp.New(cfg)

		require.NoError(t, err)
		assert.NotNil(t, client)
	})
}

// TestHTTPAntivirusClientPing tests health check
func TestHTTPAntivirusClientPing(t *testing.T) {
	t.Run("successful ping", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/health", r.URL.Path)

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy"}`))
		}))
		defer server.Close()

		cfg := antivirusHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := antivirusHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		err = client.Ping(ctx)

		require.NoError(t, err)
	})

	t.Run("ping with service unavailable", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unhealthy"}`))
		}))
		defer server.Close()

		cfg := antivirusHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := antivirusHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		err = client.Ping(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ping returned status 503")
	})

	t.Run("ping with network error", func(t *testing.T) {
		cfg := antivirusHttp.Config{
			BaseURL:        "http://invalid-host-does-not-exist:9999",
			TimeoutSeconds: 1,
		}
		client, err := antivirusHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		err = client.Ping(ctx)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ping failed")
	})

	t.Run("ping with context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("Request should be cancelled")
		}))
		defer server.Close()

		cfg := antivirusHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := antivirusHttp.New(cfg)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = client.Ping(ctx)

		assert.Error(t, err)
	})
}

// TestHTTPAntivirusClientScanReader tests file scanning
func TestHTTPAntivirusClientScanReader(t *testing.T) {
	t.Run("scan clean file", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/scan", r.URL.Path)
			assert.Contains(t, r.Header.Get("Content-Type"), "multipart/form-data")

			// Parse multipart form
			err := r.ParseMultipartForm(10 << 20)
			require.NoError(t, err)

			file, header, err := r.FormFile("file")
			require.NoError(t, err)
			defer file.Close()

			assert.Equal(t, testCleanFile, header.Filename)

			// Read file content
			content, err := io.ReadAll(file)
			require.NoError(t, err)
			assert.Equal(t, "clean file content", string(content))

			response := map[string]interface{}{
				"file_name": testCleanFile,
				"clean":     true,
				"threat":    "",
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := antivirusHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := antivirusHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		reader := strings.NewReader("clean file content")
		result, err := client.ScanReader(ctx, testCleanFile, reader)

		require.NoError(t, err)
		assert.Equal(t, testCleanFile, result.FileName)
		assert.True(t, result.Clean)
		assert.Empty(t, result.Threat)
	})

	t.Run("scan infected file", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(10 << 20)
			require.NoError(t, err)

			file, header, err := r.FormFile("file")
			require.NoError(t, err)
			defer file.Close()

			assert.Equal(t, testVirusFile, header.Filename)

			response := map[string]interface{}{
				"file_name": testVirusFile,
				"clean":     false,
				"threat":    testThreatName,
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := antivirusHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := antivirusHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		reader := strings.NewReader("malicious content")
		result, err := client.ScanReader(ctx, testVirusFile, reader)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "virus detected")
		assert.Contains(t, err.Error(), testThreatName)
		assert.Equal(t, testVirusFile, result.FileName)
		assert.False(t, result.Clean)
		assert.Equal(t, testThreatName, result.Threat)
	})

	t.Run("scan with large file", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(10 << 20)
			require.NoError(t, err)

			file, _, err := r.FormFile("file")
			require.NoError(t, err)
			defer file.Close()

			content, err := io.ReadAll(file)
			require.NoError(t, err)
			assert.Equal(t, 10000, len(content))

			response := map[string]interface{}{
				"file_name": testLargeFile,
				"clean":     true,
				"threat":    "",
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := antivirusHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := antivirusHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		largeContent := bytes.Repeat([]byte("x"), 10000)
		reader := bytes.NewReader(largeContent)
		result, err := client.ScanReader(ctx, testLargeFile, reader)

		require.NoError(t, err)
		assert.True(t, result.Clean)
	})

	t.Run("scan with empty file", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(10 << 20)
			require.NoError(t, err)

			file, _, err := r.FormFile("file")
			require.NoError(t, err)
			defer file.Close()

			content, err := io.ReadAll(file)
			require.NoError(t, err)
			assert.Equal(t, 0, len(content))

			response := map[string]interface{}{
				"file_name": "empty.txt",
				"clean":     true,
				"threat":    "",
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := antivirusHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := antivirusHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		reader := strings.NewReader("")
		result, err := client.ScanReader(ctx, "empty.txt", reader)

		require.NoError(t, err)
		assert.True(t, result.Clean)
	})

	t.Run("500 internal server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "internal error"}`))
		}))
		defer server.Close()

		cfg := antivirusHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := antivirusHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		reader := strings.NewReader("content")
		result, err := client.ScanReader(ctx, testFileName, reader)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "scan failed with status 500")
		assert.False(t, result.Clean)
		assert.Contains(t, result.Threat, "HTTP 500")
	})

	t.Run("scan with network error", func(t *testing.T) {
		cfg := antivirusHttp.Config{
			BaseURL:        "http://invalid-host:9999",
			TimeoutSeconds: 1,
		}
		client, err := antivirusHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		reader := strings.NewReader("content")
		result, err := client.ScanReader(ctx, testFileName, reader)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send request")
		assert.False(t, result.Clean)
		assert.Equal(t, "failed to send request", result.Threat)
	})

	t.Run("scan with invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`invalid json`))
		}))
		defer server.Close()

		cfg := antivirusHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := antivirusHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		reader := strings.NewReader("content")
		result, err := client.ScanReader(ctx, testFileName, reader)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode response")
		assert.False(t, result.Clean)
		assert.Equal(t, "failed to parse response", result.Threat)
	})

	t.Run("scan with context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("Request should be cancelled")
		}))
		defer server.Close()

		cfg := antivirusHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := antivirusHttp.New(cfg)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		reader := strings.NewReader("content")
		result, err := client.ScanReader(ctx, testFileName, reader)

		assert.Error(t, err)
		assert.False(t, result.Clean)
	})

	t.Run("scan with special characters in filename", func(t *testing.T) {
		specialFileName := "file with spaces & special@chars.txt"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(10 << 20)
			require.NoError(t, err)

			_, header, err := r.FormFile("file")
			require.NoError(t, err)

			assert.Equal(t, specialFileName, header.Filename)

			response := map[string]interface{}{
				"file_name": specialFileName,
				"clean":     true,
				"threat":    "",
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := antivirusHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := antivirusHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		reader := strings.NewReader("content")
		result, err := client.ScanReader(ctx, specialFileName, reader)

		require.NoError(t, err)
		assert.Equal(t, specialFileName, result.FileName)
		assert.True(t, result.Clean)
	})

	t.Run("scan with unicode filename", func(t *testing.T) {
		unicodeFileName := "测试文件.txt"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			err := r.ParseMultipartForm(10 << 20)
			require.NoError(t, err)

			_, header, err := r.FormFile("file")
			require.NoError(t, err)

			assert.Equal(t, unicodeFileName, header.Filename)

			response := map[string]interface{}{
				"file_name": unicodeFileName,
				"clean":     true,
				"threat":    "",
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := antivirusHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := antivirusHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		reader := strings.NewReader("content")
		result, err := client.ScanReader(ctx, unicodeFileName, reader)

		require.NoError(t, err)
		assert.Equal(t, unicodeFileName, result.FileName)
		assert.True(t, result.Clean)
	})

	t.Run("scan multiple threats", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"file_name": "multi-threat.exe",
				"clean":     false,
				"threat":    "Trojan.Generic, Malware.BadStuff, Virus.Test",
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := antivirusHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := antivirusHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		reader := strings.NewReader("malicious")
		result, err := client.ScanReader(ctx, "multi-threat.exe", reader)

		assert.Error(t, err)
		assert.False(t, result.Clean)
		assert.Contains(t, result.Threat, "Trojan.Generic")
		assert.Contains(t, result.Threat, "Malware.BadStuff")
	})

	t.Run("scan with timeout", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate slow scan
			time.Sleep(2 * time.Second)
			// The client timeout should fire before this
			response := map[string]interface{}{
				"file_name": testFileName,
				"clean":     true,
				"threat":    "",
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(response)
		}))
		defer server.Close()

		cfg := antivirusHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 1, // Very short timeout
		}
		client, err := antivirusHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		reader := strings.NewReader("content")
		result, err := client.ScanReader(ctx, testFileName, reader)

		assert.Error(t, err)
		assert.False(t, result.Clean)
	})
}

// TestHTTPAntivirusClientErrorResponses tests various error response formats
func TestHTTPAntivirusClientErrorResponses(t *testing.T) {
	t.Run("400 bad request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"invalid file format"}`))
		}))
		defer server.Close()

		cfg := antivirusHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := antivirusHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		reader := strings.NewReader("content")
		result, err := client.ScanReader(ctx, "test.txt", reader)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "scan failed with status 400")
		assert.False(t, result.Clean)
	})

	t.Run("413 payload too large", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			w.Write([]byte(`{"error":"file too large"}`))
		}))
		defer server.Close()

		cfg := antivirusHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := antivirusHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		largeContent := bytes.Repeat([]byte("x"), 1000000)
		reader := bytes.NewReader(largeContent)
		result, err := client.ScanReader(ctx, testLargeFile, reader)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "scan failed with status 413")
		assert.False(t, result.Clean)
	})

	t.Run("503 service unavailable", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"error":"service unavailable"}`))
		}))
		defer server.Close()

		cfg := antivirusHttp.Config{
			BaseURL:        server.URL,
			TimeoutSeconds: 30,
		}
		client, err := antivirusHttp.New(cfg)
		require.NoError(t, err)

		ctx := context.Background()
		reader := strings.NewReader("content")
		result, err := client.ScanReader(ctx, testFileName, reader)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "scan failed with status 503")
		assert.False(t, result.Clean)
	})
}

// TestHTTPAntivirusClientConcurrentScans tests concurrent scanning
func TestHTTPAntivirusClientConcurrentScans(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		err := r.ParseMultipartForm(10 << 20)
		require.NoError(t, err)

		_, header, err := r.FormFile("file")
		require.NoError(t, err)

		response := map[string]interface{}{
			"file_name": header.Filename,
			"clean":     true,
			"threat":    "",
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	cfg := antivirusHttp.Config{
		BaseURL:        server.URL,
		TimeoutSeconds: 30,
	}
	client, err := antivirusHttp.New(cfg)
	require.NoError(t, err)

	// Launch multiple concurrent scans
	numScans := 10
	done := make(chan bool, numScans)

	for i := 0; i < numScans; i++ {
		go func(index int) {
			ctx := context.Background()
			reader := strings.NewReader("content")
			fileName := fmt.Sprintf("file%d.txt", index)
			result, err := client.ScanReader(ctx, fileName, reader)

			assert.NoError(t, err)
			assert.True(t, result.Clean)

			done <- true
		}(i)
	}

	// Wait for all scans
	for i := 0; i < numScans; i++ {
		<-done
	}

	assert.Equal(t, numScans, requestCount)
}
