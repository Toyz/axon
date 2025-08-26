package axon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResponse_NewResponse(t *testing.T) {
	body := map[string]string{"message": "success"}
	resp := NewResponse(201, body)

	assert.Equal(t, 201, resp.StatusCode)
	assert.Equal(t, body, resp.Body)
}

func TestResponse_OK(t *testing.T) {
	body := map[string]string{"data": "test"}
	resp := OK(body)

	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, body, resp.Body)
}

func TestResponse_Created(t *testing.T) {
	body := map[string]string{"id": "123"}
	resp := Created(body)

	assert.Equal(t, 201, resp.StatusCode)
	assert.Equal(t, body, resp.Body)
}

func TestResponse_NoContent(t *testing.T) {
	resp := NoContent()

	assert.Equal(t, 204, resp.StatusCode)
	assert.Nil(t, resp.Body)
}

func TestResponse_BadRequest(t *testing.T) {
	resp := BadRequest("Invalid input")

	assert.Equal(t, 400, resp.StatusCode)
	assert.Equal(t, map[string]string{"error": "Invalid input"}, resp.Body)
}

func TestResponse_NotFound(t *testing.T) {
	resp := NotFound("User not found")

	assert.Equal(t, 404, resp.StatusCode)
	assert.Equal(t, map[string]string{"error": "User not found"}, resp.Body)
}

func TestResponse_InternalServerError(t *testing.T) {
	resp := InternalServerError("Database connection failed")

	assert.Equal(t, 500, resp.StatusCode)
	assert.Equal(t, map[string]string{"error": "Database connection failed"}, resp.Body)
}
