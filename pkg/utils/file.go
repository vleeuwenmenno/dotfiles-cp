package utils

import (
	"encoding/json"
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

// GetContentDiffSummary returns a simple summary of differences between two text contents
func GetContentDiffSummary(oldContent, newContent string) []string {
	var changes []string

	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	if len(oldLines) != len(newLines) {
		changes = append(changes, fmt.Sprintf("Line count: %d -> %d lines", len(oldLines), len(newLines)))
	}

	// Count changed lines (simple comparison)
	changedLines := 0
	maxLines := len(oldLines)
	if len(newLines) > maxLines {
		maxLines = len(newLines)
	}

	for i := 0; i < maxLines; i++ {
		oldLine := ""
		newLine := ""

		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}

		if oldLine != newLine {
			changedLines++
		}
	}

	if changedLines > 0 {
		changes = append(changes, fmt.Sprintf("Changed lines: %d", changedLines))
	}

	// Show size difference
	if len(oldContent) != len(newContent) {
		changes = append(changes, fmt.Sprintf("Size: %d -> %d bytes", len(oldContent), len(newContent)))
	}

	return changes
}

// GetDetailedDiff returns a detailed diff showing actual content changes
func GetDetailedDiff(oldContent, newContent string, maxLines int) []string {
	if maxLines <= 0 {
		maxLines = 20 // Default limit
	}

	var diff []string
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	// Simple line-by-line diff implementation
	maxLength := len(oldLines)
	if len(newLines) > maxLength {
		maxLength = len(newLines)
	}

	diffCount := 0
	for i := 0; i < maxLength && diffCount < maxLines; i++ {
		oldLine := ""
		newLine := ""

		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		if i < len(newLines) {
			newLine = newLines[i]
		}

		if oldLine != newLine {
			diffCount++
			lineNum := i + 1

			if oldLine != "" {
				diff = append(diff, fmt.Sprintf("- %d: %s", lineNum, oldLine))
			}
			if newLine != "" {
				diff = append(diff, fmt.Sprintf("+ %d: %s", lineNum, newLine))
			}
			if oldLine == "" {
				diff = append(diff, fmt.Sprintf("+ %d: (new line)", lineNum))
			}
			if newLine == "" {
				diff = append(diff, fmt.Sprintf("- %d: (deleted line)", lineNum))
			}
		}
	}

	if diffCount >= maxLines && maxLength > maxLines {
		diff = append(diff, fmt.Sprintf("... (showing first %d changes, %d more changes not shown)", maxLines, maxLength-maxLines))
	}

	return diff
}

// ToJSONString converts any value to a pretty-printed JSON string
func ToJSONString(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to marshal JSON: %s"}`, err.Error())
	}
	return string(data)
}

// DirExists checks if a directory exists
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
