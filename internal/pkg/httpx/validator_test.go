package httpx

import (
	"bytes"
	"fmt"
	"mime"
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testInput struct {
	File *multipart.FileHeader `validate:"omitempty,max_size=10485760"`
}

func TestMaxSizeValidator(t *testing.T) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	p, _ := w.CreateFormFile("file", "big.png")
	p.Write(make([]byte, 11<<20))
	w.Close()

	_, params, _ := mime.ParseMediaType(w.FormDataContentType())
	r := multipart.NewReader(&body, params["boundary"])
	form, _ := r.ReadForm(32 << 20)
	fh := form.File["file"][0]
	fmt.Printf("File size: %d\n", fh.Size)

	v := NewValidator()
	err := v.Struct(&testInput{File: fh})
	assert.Error(t, err)
}
