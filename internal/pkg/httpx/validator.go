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
	fh, ok := fl.Field().Interface().(*multipart.FileHeader)
	if !ok || fh == nil {
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
	return fh.Size <= maxBytes
}
