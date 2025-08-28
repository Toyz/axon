package controllers

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

// axon::controller -Prefix=/files
type FileController struct{}

// ServeStaticFiles handles all requests under /files/* path
// This demonstrates wildcard route functionality
// axon::route GET /{*}
func (c *FileController) ServeStaticFiles(ctx echo.Context, wildcardPath string) (map[string]interface{}, error) {
	// Basic security check to prevent directory traversal
	if strings.Contains(wildcardPath, "..") {
		return nil, fmt.Errorf("invalid path")
	}

	// Determine content type based on file extension
	ext := filepath.Ext(wildcardPath)
	var contentType string
	switch ext {
	case ".css":
		contentType = "text/css"
	case ".js":
		contentType = "application/javascript"
	case ".html":
		contentType = "text/html"
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".gif":
		contentType = "image/gif"
	default:
		contentType = "text/plain"
	}

	// Return file information
	response := map[string]interface{}{
		"message":        "File served successfully",
		"requested_path": wildcardPath,
		"full_path":      fmt.Sprintf("/files/%s", wildcardPath),
		"content_type":   contentType,
		"file_exists":    true, // In real implementation, check if file exists
	}

	return response, nil
}

// UploadFile handles file uploads (more specific route)
// This demonstrates that specific routes still work alongside wildcards
// axon::route POST /upload
func (c *FileController) UploadFile() (map[string]interface{}, error) {
	return map[string]interface{}{
		"message": "File upload endpoint",
		"note":    "This specific route takes precedence over the wildcard route",
	}, nil
}

// GetFileInfo shows information about a specific file
// This also demonstrates specific routes working with wildcards
// axon::route GET /finfo/{filename:string}
func (c *FileController) GetFileInfo(filename string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"message":  "File info endpoint",
		"filename": filename,
		"note":     "This specific parameterized route also takes precedence over wildcard",
	}, nil
}
