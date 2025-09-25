package fileops

import (
	"path/filepath"

	"github.com/toyz/axon/internal/errors"
)

// ErrorWrapper provides consistent error wrapping for file operations
type ErrorWrapper struct{}

// NewErrorWrapper creates a new ErrorWrapper instance
func NewErrorWrapper() *ErrorWrapper {
	return &ErrorWrapper{}
}

// WrapFileReadError wraps file reading errors with context
func (ew *ErrorWrapper) WrapFileReadError(filePath string, err error) error {
	return errors.WrapFileSystemError("read", filePath, err)
}

// WrapFileWriteError wraps file writing errors with context
func (ew *ErrorWrapper) WrapFileWriteError(filePath string, err error) error {
	return errors.WrapFileSystemError("write", filePath, err)
}

// WrapParseError wraps parsing errors with context
func (ew *ErrorWrapper) WrapParseError(filePath string, err error) error {
	return errors.WrapParseError(filepath.Base(filePath), err)
}

// WrapDirectoryReadError wraps directory reading errors with context
func (ew *ErrorWrapper) WrapDirectoryReadError(dirPath string, err error) error {
	return errors.WrapFileSystemError("read directory", dirPath, err)
}

// WrapPathResolutionError wraps path resolution errors with context
func (ew *ErrorWrapper) WrapPathResolutionError(path string, err error) error {
	return errors.WrapFileSystemError("resolve path", path, err)
}

// WrapFileRemovalError wraps file removal errors with context
func (ew *ErrorWrapper) WrapFileRemovalError(filePath string, err error) error {
	return errors.WrapFileSystemError("remove", filePath, err)
}

// WrapFileCheckError wraps file existence check errors with context
func (ew *ErrorWrapper) WrapFileCheckError(filePath string, err error) error {
	return errors.WrapFileSystemError("check", filePath, err)
}

// WrapDirectoryCleanError wraps directory cleaning errors with context
func (ew *ErrorWrapper) WrapDirectoryCleanError(dirPath string, err error) error {
	return errors.WrapFileSystemError("clean directory", dirPath, err)
}

// WrapGoFileCheckError wraps Go file check errors with context
func (ew *ErrorWrapper) WrapGoFileCheckError(dirPath string, err error) error {
	return errors.WrapFileSystemError("check Go files", dirPath, err)
}