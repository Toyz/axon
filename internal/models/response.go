package models

// Response represents a custom HTTP response with status code and body
type Response struct {
	StatusCode int // HTTP status code
	Body       any // response body (will be JSON marshaled)
}

// NewResponse creates a new Response with the given status code and body
func NewResponse(statusCode int, body any) *Response {
	return &Response{
		StatusCode: statusCode,
		Body:       body,
	}
}