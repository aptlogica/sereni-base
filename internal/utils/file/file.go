package file

import (
	"io"
	"os"
)

// ReadFile reads the content of a file and returns it as a byte slice.
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// WriteFile writes data to a file, creating it if it does not exist.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// AppendToFile appends data to a file, creating it if it does not exist.
func AppendToFile(path string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

// DeleteFile removes the specified file from the filesystem.
func DeleteFile(path string) error {
	return os.Remove(path)
}

// FileExists checks if a file exists at the given path.
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// CopyFile copies a file from src to dst. If dst exists, it will be overwritten.
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return err
	}
	return dstFile.Sync()
}

// MoveFile moves (renames) a file from src to dst.
func MoveFile(src, dst string) error {
	return os.Rename(src, dst)
}

// GetFileInfo returns the FileInfo structure describing the file.
func GetFileInfo(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

// CreateDir creates a directory (and any necessary parents) at the specified path.
func CreateDir(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// CreateDirIfNotExists creates a directory (and any necessary parents) only if it does not already exist.
func CreateDirIfNotExists(path string, perm os.FileMode) error {
	if DirExists(path) {
		return nil
	}
	return os.MkdirAll(path, perm)
}

// DirExists checks if a directory exists at the given path.
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// DeleteDir removes a directory and its contents recursively.
func DeleteDir(path string) error {
	return os.RemoveAll(path)
}

// MoveDir moves (renames) a directory from src to dst.
func MoveDir(src, dst string) error {
	return os.Rename(src, dst)
}
