package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

func NewTestContext(method, target string, body any) (*httptest.ResponseRecorder, *gin.Context) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	var reqBody io.Reader
	switch v := body.(type) {
	case nil:
		reqBody = nil
	case string:
		reqBody = bytes.NewBufferString(v)
	case []byte:
		reqBody = bytes.NewBuffer(v)
	default:
		b, _ := json.Marshal(v)
		reqBody = bytes.NewBuffer(b)
	}

	req := httptest.NewRequest(method, target, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	c.Request = req
	return w, c
}
