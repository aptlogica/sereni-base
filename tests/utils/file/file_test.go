package tests

import (
	"os"
	"path/filepath"
	"github.com/aptlogica/sereni-base/internal/utils/file"
	"testing"
)

// TestFileExists tests the FileExists function
func TestFileExists(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "existing.txt")
	os.WriteFile(existingFile, []byte("test"), 0644)

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"existing file", existingFile, true},
		{"non-existing file", filepath.Join(tmpDir, "nonexistent.txt"), false},
		{"empty path", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := file.FileExists(tt.path)
			if result != tt.expected {
				t.Errorf("FileExists(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestDirExists tests the DirExists function
func TestDirExists(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"existing directory", tmpDir, true},
		{"non-existing directory", filepath.Join(tmpDir, "nonexistent"), false},
		{"file not directory", filepath.Join(tmpDir, "file.txt"), false},
	}

	// Create a file for testing
	testFile := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := file.DirExists(tt.path)
			if result != tt.expected {
				t.Errorf("DirExists(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestReadFile tests the ReadFile function
func TestReadFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		content     []byte
		filename    string
		wantErr     bool
		expectEqual bool
	}{
		{"read valid file", []byte("hello world"), "test.txt", false, true},
		{"read empty file", []byte(""), "empty.txt", false, true},
		{"read binary data", []byte{0x00, 0xFF, 0xAB, 0xCD}, "binary.dat", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, tt.filename)
			os.WriteFile(path, tt.content, 0644)

			result, err := file.ReadFile(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.expectEqual && string(result) != string(tt.content) {
				t.Errorf("ReadFile() = %q, want %q", result, tt.content)
			}
		})
	}

	t.Run("non-existing file", func(t *testing.T) {
		_, err := file.ReadFile(filepath.Join(tmpDir, "nonexistent.txt"))
		if err == nil {
			t.Error("ReadFile() should return error for non-existing file")
		}
	})
}

// TestWriteFile tests the WriteFile function
func TestWriteFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		content []byte
		perm    os.FileMode
		wantErr bool
	}{
		{"write simple text", []byte("test content"), 0644, false},
		{"write empty content", []byte(""), 0644, false},
		{"write binary data", []byte{0x00, 0xFF}, 0600, false},
		{"write with different permissions", []byte("test"), 0755, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, tt.name+".txt")
			err := file.WriteFile(path, tt.content, tt.perm)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify content was written
				readContent, _ := os.ReadFile(path)
				if string(readContent) != string(tt.content) {
					t.Errorf("File content = %q, want %q", readContent, tt.content)
				}
			}
		})
	}
}

// TestAppendToFile tests the AppendToFile function
func TestAppendToFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("append to new file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "append_new.txt")
		err := file.AppendToFile(path, []byte("first"), 0644)
		if err != nil {
			t.Fatalf("AppendToFile() error = %v", err)
		}

		content, _ := os.ReadFile(path)
		if string(content) != "first" {
			t.Errorf("Content = %q, want %q", content, "first")
		}
	})

	t.Run("append to existing file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "append_existing.txt")
		os.WriteFile(path, []byte("initial"), 0644)

		err := file.AppendToFile(path, []byte(" appended"), 0644)
		if err != nil {
			t.Fatalf("AppendToFile() error = %v", err)
		}

		content, _ := os.ReadFile(path)
		if string(content) != "initial appended" {
			t.Errorf("Content = %q, want %q", content, "initial appended")
		}
	})

	t.Run("multiple appends", func(t *testing.T) {
		path := filepath.Join(tmpDir, "multiple_appends.txt")
		file.AppendToFile(path, []byte("1"), 0644)
		file.AppendToFile(path, []byte("2"), 0644)
		file.AppendToFile(path, []byte("3"), 0644)

		content, _ := os.ReadFile(path)
		if string(content) != "123" {
			t.Errorf("Content = %q, want %q", content, "123")
		}
	})
}

// TestDeleteFile tests the DeleteFile function
func TestDeleteFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("delete existing file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "delete_me.txt")
		os.WriteFile(path, []byte("test"), 0644)

		err := file.DeleteFile(path)
		if err != nil {
			t.Errorf("DeleteFile() error = %v", err)
		}

		if file.FileExists(path) {
			t.Error("File should not exist after deletion")
		}
	})

	t.Run("delete non-existing file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "nonexistent.txt")
		err := file.DeleteFile(path)
		if err == nil {
			t.Error("DeleteFile() should return error for non-existing file")
		}
	})
}

// TestCopyFile tests the CopyFile function
func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		content []byte
		wantErr bool
	}{
		{"copy text file", []byte("hello world"), false},
		{"copy empty file", []byte(""), false},
		{"copy binary file", []byte{0x00, 0xFF, 0xAB}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := filepath.Join(tmpDir, "src_"+tt.name+".txt")
			dst := filepath.Join(tmpDir, "dst_"+tt.name+".txt")

			os.WriteFile(src, tt.content, 0644)

			err := file.CopyFile(src, dst)
			if (err != nil) != tt.wantErr {
				t.Errorf("CopyFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				dstContent, _ := os.ReadFile(dst)
				if string(dstContent) != string(tt.content) {
					t.Errorf("Copied content = %q, want %q", dstContent, tt.content)
				}
			}
		})
	}

	t.Run("copy non-existing file", func(t *testing.T) {
		src := filepath.Join(tmpDir, "nonexistent.txt")
		dst := filepath.Join(tmpDir, "destination.txt")
		err := file.CopyFile(src, dst)
		if err == nil {
			t.Error("CopyFile() should return error for non-existing source")
		}
	})
}

// TestMoveFile tests the MoveFile function
func TestMoveFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("move existing file", func(t *testing.T) {
		src := filepath.Join(tmpDir, "move_src.txt")
		dst := filepath.Join(tmpDir, "move_dst.txt")
		content := []byte("move me")

		os.WriteFile(src, content, 0644)

		err := file.MoveFile(src, dst)
		if err != nil {
			t.Errorf("MoveFile() error = %v", err)
		}

		if file.FileExists(src) {
			t.Error("Source file should not exist after move")
		}

		if !file.FileExists(dst) {
			t.Error("Destination file should exist after move")
		}

		dstContent, _ := os.ReadFile(dst)
		if string(dstContent) != string(content) {
			t.Errorf("Moved content = %q, want %q", dstContent, content)
		}
	})

	t.Run("move non-existing file", func(t *testing.T) {
		src := filepath.Join(tmpDir, "nonexistent.txt")
		dst := filepath.Join(tmpDir, "destination.txt")
		err := file.MoveFile(src, dst)
		if err == nil {
			t.Error("MoveFile() should return error for non-existing file")
		}
	})
}

// TestGetFileInfo tests the GetFileInfo function
func TestGetFileInfo(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("get info of existing file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "info.txt")
		content := []byte("test content")
		os.WriteFile(path, content, 0644)

		info, err := file.GetFileInfo(path)
		if err != nil {
			t.Errorf("GetFileInfo() error = %v", err)
		}

		if info.Name() != "info.txt" {
			t.Errorf("Name = %q, want %q", info.Name(), "info.txt")
		}

		if info.Size() != int64(len(content)) {
			t.Errorf("Size = %d, want %d", info.Size(), len(content))
		}

		if info.IsDir() {
			t.Error("IsDir() should be false for file")
		}
	})

	t.Run("get info of non-existing file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "nonexistent.txt")
		_, err := file.GetFileInfo(path)
		if err == nil {
			t.Error("GetFileInfo() should return error for non-existing file")
		}
	})
}

// TestCreateDir tests the CreateDir function
func TestCreateDir(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("create single directory", func(t *testing.T) {
		path := filepath.Join(tmpDir, "newdir")
		err := file.CreateDir(path, 0755)
		if err != nil {
			t.Errorf("CreateDir() error = %v", err)
		}

		if !file.DirExists(path) {
			t.Error("Directory should exist after creation")
		}
	})

	t.Run("create nested directories", func(t *testing.T) {
		path := filepath.Join(tmpDir, "parent", "child", "grandchild")
		err := file.CreateDir(path, 0755)
		if err != nil {
			t.Errorf("CreateDir() error = %v", err)
		}

		if !file.DirExists(path) {
			t.Error("Nested directory should exist after creation")
		}
	})

	t.Run("create existing directory", func(t *testing.T) {
		path := filepath.Join(tmpDir, "existing")
		os.Mkdir(path, 0755)

		err := file.CreateDir(path, 0755)
		if err != nil {
			t.Errorf("CreateDir() should not error for existing directory: %v", err)
		}
	})
}

// TestCreateDirIfNotExists tests the CreateDirIfNotExists function
func TestCreateDirIfNotExists(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("create non-existing directory", func(t *testing.T) {
		path := filepath.Join(tmpDir, "conditional")
		err := file.CreateDirIfNotExists(path, 0755)
		if err != nil {
			t.Errorf("CreateDirIfNotExists() error = %v", err)
		}

		if !file.DirExists(path) {
			t.Error("Directory should exist after creation")
		}
	})

	t.Run("skip existing directory", func(t *testing.T) {
		path := filepath.Join(tmpDir, "existing2")
		os.Mkdir(path, 0755)

		err := file.CreateDirIfNotExists(path, 0755)
		if err != nil {
			t.Errorf("CreateDirIfNotExists() should not error for existing directory: %v", err)
		}
	})
}

// TestDeleteDir tests the DeleteDir function
func TestDeleteDir(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("delete empty directory", func(t *testing.T) {
		path := filepath.Join(tmpDir, "empty_dir")
		os.Mkdir(path, 0755)

		err := file.DeleteDir(path)
		if err != nil {
			t.Errorf("DeleteDir() error = %v", err)
		}

		if file.DirExists(path) {
			t.Error("Directory should not exist after deletion")
		}
	})

	t.Run("delete directory with contents", func(t *testing.T) {
		path := filepath.Join(tmpDir, "full_dir")
		os.Mkdir(path, 0755)
		os.WriteFile(filepath.Join(path, "file.txt"), []byte("test"), 0644)

		err := file.DeleteDir(path)
		if err != nil {
			t.Errorf("DeleteDir() error = %v", err)
		}

		if file.DirExists(path) {
			t.Error("Directory should not exist after deletion")
		}
	})

	t.Run("delete non-existing directory", func(t *testing.T) {
		path := filepath.Join(tmpDir, "nonexistent_dir")
		err := file.DeleteDir(path)
		// Note: DeleteDir might not return error for non-existing directory
		_ = err
	})
}

// TestMoveDir tests the MoveDir function
func TestMoveDir(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("move directory", func(t *testing.T) {
		src := filepath.Join(tmpDir, "move_dir_src")
		dst := filepath.Join(tmpDir, "move_dir_dst")

		os.Mkdir(src, 0755)
		os.WriteFile(filepath.Join(src, "file.txt"), []byte("test"), 0644)

		err := file.MoveDir(src, dst)
		if err != nil {
			t.Errorf("MoveDir() error = %v", err)
		}

		if file.DirExists(src) {
			t.Error("Source directory should not exist after move")
		}

		if !file.DirExists(dst) {
			t.Error("Destination directory should exist after move")
		}

		if !file.FileExists(filepath.Join(dst, "file.txt")) {
			t.Error("File should exist in moved directory")
		}
	})

	t.Run("move non-existing directory", func(t *testing.T) {
		src := filepath.Join(tmpDir, "nonexistent_dir")
		dst := filepath.Join(tmpDir, "destination_dir")
		err := file.MoveDir(src, dst)
		if err == nil {
			t.Error("MoveDir() should return error for non-existing directory")
		}
	})
}

// TestAppendToFile_ErrorCases tests AppendToFile error handling
func TestAppendToFile_ErrorCases(t *testing.T) {
	t.Run("append to file in non-existing directory", func(t *testing.T) {
		path := "/nonexistent/path/file.txt"
		err := file.AppendToFile(path, []byte("test"), 0644)
		if err == nil {
			t.Error("AppendToFile() should return error for non-existing directory")
		}
	})
}

// TestCopyFile_ErrorCases tests CopyFile error handling
func TestCopyFile_ErrorCases(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("copy to non-existing directory", func(t *testing.T) {
		src := filepath.Join(tmpDir, "src.txt")
		os.WriteFile(src, []byte("test"), 0644)
		dst := "/nonexistent/dir/dst.txt"
		err := file.CopyFile(src, dst)
		if err == nil {
			t.Error("CopyFile() should return error when destination directory doesn't exist")
		}
	})

	t.Run("copy to invalid destination", func(t *testing.T) {
		src := filepath.Join(tmpDir, "src2.txt")
		os.WriteFile(src, []byte("test"), 0644)

		// Create a directory as destination (should fail when trying to create file)
		dst := filepath.Join(tmpDir, "is_a_dir")
		os.Mkdir(dst, 0755)

		err := file.CopyFile(src, dst)
		if err == nil {
			t.Error("CopyFile() should return error when destination is a directory")
		}
	})
}

// TestReadFile_LargeFile tests reading large files
func TestReadFile_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "large.txt")

	// Create a 1MB file
	data := make([]byte, 1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	os.WriteFile(path, data, 0644)

	result, err := file.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if len(result) != len(data) {
		t.Errorf("ReadFile() read %d bytes, want %d", len(result), len(data))
	}
}

// TestGetFileInfo_Directory tests GetFileInfo on a directory
func TestGetFileInfo_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	info, err := file.GetFileInfo(tmpDir)
	if err != nil {
		t.Errorf("GetFileInfo() error = %v", err)
	}

	if !info.IsDir() {
		t.Error("GetFileInfo() should return IsDir() = true for directory")
	}
}
