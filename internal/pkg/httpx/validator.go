package httpx

import (
	"mime/multipart"
	"strconv"

	"github.com/go-playground/validator/v10"
)

func NewValidator() *validator.Validate {
	v := validator.New()
	_ = v.RegisterValidation("max_size", maxFileSizeValidator)
	return v
}

func maxFileSizeValidator(fl validator.FieldLevel) bool {
	var size int64
	switch v := fl.Field().Interface().(type) {
	case *multipart.FileHeader:
		if v == nil {
			return true
		}
		size = v.Size
	case multipart.FileHeader:
		size = v.Size
	default:
		return true
	}
	param := fl.Param()
	if param == "" {
		return false
	}
	maxBytes, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		return false
	}
	return size <= maxBytes
}
