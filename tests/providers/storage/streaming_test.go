package storage_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	storageHttp "github.com/aptlogica/sereni-base/internal/providers/storage/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The upload is streamed as multipart; the storage service must still receive the file and its path.
func TestUploadStreamsTheFileAsMultipart(t *testing.T) {
	content := bytes.Repeat([]byte("0123456789abcdef"), 1<<18) // 4 MiB
	var gotPath string
	var got []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseMultipartForm(1<<20))
		gotPath = r.FormValue("path")
		file, _, err := r.FormFile("file")
		require.NoError(t, err)
		got, _ = io.ReadAll(file)
		_, _ = w.Write([]byte(`{"message":"ok","path":"exports/x.zip","url":"u"}`))
	}))
	defer srv.Close()

	client, err := storageHttp.New(storageHttp.Config{BaseURL: srv.URL, TimeoutSeconds: 30})
	require.NoError(t, err)
	res, err := client.Upload(context.Background(), "exports/x.zip", bytes.NewReader(content), int64(len(content)), "application/zip")
	require.NoError(t, err)
	assert.Equal(t, "exports/x.zip", res.Path)
	assert.Equal(t, "exports/x.zip", gotPath)
	assert.True(t, bytes.Equal(content, got), "the service must receive the whole file")
}

// A reader that fails part-way is an error for the caller, not a half-uploaded success.
func TestUploadReportsAFailingReader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseMultipartForm(1 << 20)
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))
	defer srv.Close()
	client, err := storageHttp.New(storageHttp.Config{BaseURL: srv.URL, TimeoutSeconds: 30})
	require.NoError(t, err)

	_, err = client.Upload(context.Background(), "a.zip", io.MultiReader(bytes.NewReader([]byte("abc")), failingReader{}), 10, "application/zip")
	assert.Error(t, err)
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
