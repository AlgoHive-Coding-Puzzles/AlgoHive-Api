package testutils

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// NewTestContext builds a gin.Context and its ResponseRecorder for a request
// with an optional JSON body. Pass a nil body for requests without one.
func NewTestContext(t *testing.T, method, path string, body interface{}) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var req *http.Request
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		req = httptest.NewRequest(method, path, bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	c.Request = req

	return c, w
}

// SetAuthCookie attaches an auth_token cookie to the context's request,
// mimicking an authenticated browser session.
func SetAuthCookie(c *gin.Context, token string) {
	c.Request.AddCookie(&http.Cookie{Name: "auth_token", Value: token})
}

// SetAuthHeader attaches a Bearer Authorization header to the request.
func SetAuthHeader(c *gin.Context, token string) {
	c.Request.Header.Set("Authorization", "Bearer "+token)
}

// DecodeJSON unmarshals a recorder body into target, failing the test on error.
func DecodeJSON(t *testing.T, w *httptest.ResponseRecorder, target interface{}) {
	t.Helper()
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), target))
}
