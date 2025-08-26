// Package axon provides public APIs for the Axon Framework
package axon

// Response represents an HTTP response with custom status code, headers, and body
// This struct should be used as a return type from route handlers when
// you need full control over the HTTP response.
//
// Example usage:
//
//	func (c *UserController) CreateUser(user User) (*axon.Response, error) {
//	    // ... create user logic ...
//	    return &axon.Response{
//	        StatusCode: 201,
//	        Body:       createdUser,
//	        Headers: map[string]string{
//	            "Location": "/users/123",
//	            "X-Custom-Header": "value",
//	        },
//	    }, nil
//	}
type Response struct {
	// StatusCode is the HTTP status code to return (e.g., 200, 201, 404, 500)
	StatusCode int `json:"-"`

	// Body is the response body that will be JSON-encoded and sent to the client
	Body interface{} `json:"body,omitempty"`

	// Headers contains HTTP headers to set on the response
	Headers map[string]string `json:"-"`

	// ContentType overrides the default "application/json" content type
	ContentType string `json:"-"`

	// Cookies contains cookies to set on the response
	Cookies []*Cookie `json:"-"`
}

// Cookie represents an HTTP cookie
type Cookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Path     string `json:"path,omitempty"`
	Domain   string `json:"domain,omitempty"`
	MaxAge   int    `json:"max_age,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
	HttpOnly bool   `json:"http_only,omitempty"`
	SameSite string `json:"same_site,omitempty"` // "Strict", "Lax", "None"
}

// NewResponse creates a new Response with the specified status code and body
func NewResponse(statusCode int, body interface{}) *Response {
	return &Response{
		StatusCode: statusCode,
		Body:       body,
		Headers:    make(map[string]string),
		Cookies:    make([]*Cookie, 0),
	}
}

// NewResponseWithHeaders creates a new Response with status code, body, and headers
func NewResponseWithHeaders(statusCode int, body interface{}, headers map[string]string) *Response {
	return &Response{
		StatusCode: statusCode,
		Body:       body,
		Headers:    headers,
		Cookies:    make([]*Cookie, 0),
	}
}

// OK creates a 200 OK response with the given body
func OK(body interface{}) *Response {
	return NewResponse(200, body)
}

// Created creates a 201 Created response with the given body
func Created(body interface{}) *Response {
	return NewResponse(201, body)
}

// NoContent creates a 204 No Content response
func NoContent() *Response {
	return NewResponse(204, nil)
}

// BadRequest creates a 400 Bad Request response with the given error message
func BadRequest(message string) *Response {
	return NewResponse(400, map[string]string{"error": message})
}

// NotFound creates a 404 Not Found response with the given error message
func NotFound(message string) *Response {
	return NewResponse(404, map[string]string{"error": message})
}

// InternalServerError creates a 500 Internal Server Error response with the given error message
func InternalServerError(message string) *Response {
	return NewResponse(500, map[string]string{"error": message})
}

// Response methods for fluent API

// WithHeader adds a header to the response
func (r *Response) WithHeader(key, value string) *Response {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	r.Headers[key] = value
	return r
}

// WithHeaders adds multiple headers to the response
func (r *Response) WithHeaders(headers map[string]string) *Response {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	for k, v := range headers {
		r.Headers[k] = v
	}
	return r
}

// WithContentType sets the content type of the response
func (r *Response) WithContentType(contentType string) *Response {
	r.ContentType = contentType
	return r
}

// WithCookie adds a cookie to the response
func (r *Response) WithCookie(cookie *Cookie) *Response {
	if r.Cookies == nil {
		r.Cookies = make([]*Cookie, 0)
	}
	r.Cookies = append(r.Cookies, cookie)
	return r
}

// WithSimpleCookie adds a simple cookie with just name and value
func (r *Response) WithSimpleCookie(name, value string) *Response {
	return r.WithCookie(&Cookie{
		Name:  name,
		Value: value,
	})
}

// WithSecureCookie adds a secure, HTTP-only cookie
func (r *Response) WithSecureCookie(name, value, path string, maxAge int) *Response {
	return r.WithCookie(&Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		Secure:   true,
		HttpOnly: true,
		SameSite: "Strict",
	})
}

// Convenience constructors with headers

// CreatedWithLocation creates a 201 Created response with Location header
func CreatedWithLocation(body interface{}, location string) *Response {
	return NewResponse(201, body).WithHeader("Location", location)
}

// RedirectTo creates a 302 Found redirect response
func RedirectTo(location string) *Response {
	return NewResponse(302, nil).WithHeader("Location", location)
}

// RedirectPermanent creates a 301 Moved Permanently redirect response
func RedirectPermanent(location string) *Response {
	return NewResponse(301, nil).WithHeader("Location", location)
}

// WithCacheControl creates a response with Cache-Control header
func (r *Response) WithCacheControl(directive string) *Response {
	return r.WithHeader("Cache-Control", directive)
}

// WithETag creates a response with ETag header
func (r *Response) WithETag(etag string) *Response {
	return r.WithHeader("ETag", etag)
}
