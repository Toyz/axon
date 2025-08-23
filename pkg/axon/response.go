// Package axon provides public APIs for the Axon Framework
package axon

// Response represents an HTTP response with custom status code and body
// This struct should be used as a return type from route handlers when
// you need to control the HTTP status code and response body.
//
// Example usage:
//   func (c *UserController) CreateUser(user User) (*axon.Response, error) {
//       // ... create user logic ...
//       return &axon.Response{
//           StatusCode: 201,
//           Body:       createdUser,
//       }, nil
//   }
type Response struct {
	// StatusCode is the HTTP status code to return (e.g., 200, 201, 404, 500)
	StatusCode int `json:"-"`
	
	// Body is the response body that will be JSON-encoded and sent to the client
	Body interface{} `json:"body,omitempty"`
}

// NewResponse creates a new Response with the specified status code and body
func NewResponse(statusCode int, body interface{}) *Response {
	return &Response{
		StatusCode: statusCode,
		Body:       body,
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