package axon

import (
	"context"
	"time"
)

// WebServerInterface defines the contract for web server implementations
type WebServerInterface interface {
	// Route registration
	RegisterRoute(method string, path AxonPath, handler HandlerFunc, middlewares ...MiddlewareFunc)
	RegisterGroup(prefix string) RouteGroup

	// Global middleware
	Use(middleware MiddlewareFunc)

	// Server lifecycle
	Start(addr string) error
	Stop(ctx context.Context) error

	// Server information
	Name() string
}

// RouteGroup represents a group of routes with a common prefix
type RouteGroup interface {
	RegisterRoute(method string, path AxonPath, handler HandlerFunc, middlewares ...MiddlewareFunc)
	Use(middleware MiddlewareFunc)
	Group(prefix string) RouteGroup
}

// RequestContext provides a framework-agnostic interface for handling HTTP requests
type RequestContext interface {
	// Request data
	Method() string
	Path() string
	RealIP() string

	// Parameters
	Param(key string) string
	ParamNames() []string
	ParamValues() []string
	SetParam(name, value string)

	// Query parameters
	QueryParam(key string) string
	QueryParams() map[string][]string
	QueryString() string

	// Headers
	Request() RequestInterface
	Response() ResponseInterface

	// Body handling
	Bind(i interface{}) error
	Validate(i interface{}) error

	// Context data
	Get(key string) interface{}
	Set(key string, val interface{})

	// Request body
	FormValue(name string) string
	FormParams() (map[string][]string, error)
	FormFile(name string) (FileHeader, error)
	MultipartForm() (MultipartForm, error)
}

// RequestInterface provides access to the underlying request
type RequestInterface interface {
	Header(key string) string
	SetHeader(key, value string)
	Body() []byte
	ContentLength() int64
	ContentType() string
	Cookies() []AxonCookie
	Cookie(name string) (AxonCookie, error)
}

// ResponseInterface provides response writing capabilities
type ResponseInterface interface {
	// Status
	Status() int
	SetStatus(code int)

	// Headers
	Header(key string) string
	SetHeader(key, value string)

	// Content
	JSON(code int, i interface{}) error
	JSONPretty(code int, i interface{}, indent string) error
	String(code int, s string) error
	HTML(code int, html string) error
	Blob(code int, contentType string, b []byte) error
	Stream(code int, contentType string, r interface{}) error

	// Cookies
	SetCookie(cookie AxonCookie)

	// Response data
	Size() int64
	Written() bool
	Writer() interface{} // Framework-specific writer
}

// HandlerFunc defines the signature for HTTP handlers
type HandlerFunc func(RequestContext) error

// MiddlewareFunc defines the signature for middleware
type MiddlewareFunc func(HandlerFunc) HandlerFunc

// AxonCookie represents an HTTP cookie
type AxonCookie struct {
	Name     string
	Value    string
	Path     string
	Domain   string
	Expires  time.Time
	MaxAge   int
	Secure   bool
	HttpOnly bool
	SameSite SameSiteMode
}

// SameSiteMode defines cookie SameSite attribute modes
type SameSiteMode int

const (
	SameSiteDefaultMode SameSiteMode = iota
	SameSiteLaxMode
	SameSiteStrictMode
	SameSiteNoneMode
)

// FileHeader represents an uploaded file
type FileHeader interface {
	Filename() string
	Header() map[string][]string
	Size() int64
	Open() (interface{}, error) // Returns framework-specific file
}

// MultipartForm represents a parsed multipart form
type MultipartForm interface {
	Value() map[string][]string
	File() map[string][]FileHeader
}

// HTTPError represents an HTTP error with status code and message
type HTTPError struct {
	Code     int         `json:"code"`
	Message  interface{} `json:"message"`
	Internal error       `json:"-"` // Stores the error returned by an external dependency
}

// Error makes HTTPError implement the error interface
func (he *HTTPError) Error() string {
	if he.Internal != nil {
		return he.Internal.Error()
	}
	return he.Message.(string)
}

// NewHTTPError creates a new HTTPError instance
func NewHTTPError(code int, message ...interface{}) *HTTPError {
	he := &HTTPError{Code: code}
	if len(message) > 0 {
		he.Message = message[0]
	} else {
		he.Message = StatusText(code)
	}
	if len(message) > 1 {
		if err, ok := message[1].(error); ok {
			he.Internal = err
		}
	}
	return he
}

// StatusText returns a text for the HTTP status code
func StatusText(code int) string {
	switch code {
	case 200:
		return "OK"
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 500:
		return "Internal Server Error"
	default:
		return "Unknown"
	}
}