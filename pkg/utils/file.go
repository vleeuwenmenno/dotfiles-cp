package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileExists checks if a file or directory exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// IsDirectory checks if the given path is a directory
func IsDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsFile checks if the given path is a regular file
func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// EnsureDir creates a directory and all necessary parent directories
func EnsureDir(path string) error {
	if FileExists(path) {
		if IsDirectory(path) {
			return nil
		}
		return fmt.Errorf("path exists but is not a directory: %s", path)
	}
	return os.MkdirAll(path, 0755)
}

// ExpandPath expands ~ to the user's home directory and resolves relative paths
func ExpandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, path[2:]), nil
	}

	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, path[1:]), nil
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	return absPath, nil
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err := EnsureDir(dstDir); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy file contents
	_, err = sourceFile.WriteTo(destFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// BackupFile creates a backup of a file by appending .backup to the filename
func BackupFile(path string) error {
	if !FileExists(path) {
		return fmt.Errorf("file does not exist: %s", path)
	}

	backupPath := path + ".backup"
	return CopyFile(path, backupPath)
}

// RemoveBackup removes a backup file
func RemoveBackup(path string) error {
	backupPath := path + ".backup"
	if FileExists(backupPath) {
		return os.Remove(backupPath)
	}
	return nil
}

// CreateSymlink creates a symbolic link from src to dst
func CreateSymlink(src, dst string) error {
	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err := EnsureDir(dstDir); err != nil {
		return err
	}

	// Remove existing file/symlink if it exists
	if FileExists(dst) {
		if err := os.Remove(dst); err != nil {
			return err
		}
	}

	return os.Symlink(src, dst)
}

// IsSymlink checks if the given path is a symbolic link
func IsSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

// GetSymlinkTarget returns the target of a symbolic link
func GetSymlinkTarget(path string) (string, error) {
	if !IsSymlink(path) {
		return "", fmt.Errorf("path is not a symbolic link: %s", path)
	}
	return os.Readlink(path)
}

// SanitizePath cleans and normalizes a file path
func SanitizePath(path string) string {
	// Clean the path (removes redundant separators and . elements)
	cleaned := filepath.Clean(path)

	// Convert to forward slashes for consistency (Go handles this automatically on Windows)
	return filepath.ToSlash(cleaned)
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// WriteFile writes data to a file, creating directories as needed
func WriteFile(path string, data []byte, perm os.FileMode) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return err
	}

	return os.WriteFile(path, data, perm)
}

// ReadFile reads the contents of a file
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
