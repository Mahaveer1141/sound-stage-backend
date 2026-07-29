package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

type PaginatedResponse struct {
	Success    bool       `json:"success"`
	Message    string     `json:"message,omitempty"`
	Data       any        `json:"data,omitempty"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	Page       int `json:"page"`
	TotalCount int `json:"totalCount"`
	TotalPages int `json:"totalPages"`
}

func SuccessResponse(c *gin.Context, statusCode int, message string, data any) {
	c.JSON(statusCode, Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func ErrorResponse(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, Response{
		Success: false,
		Message: message,
	})
}

func PaginatedSuccessResponse(c *gin.Context, message string, data any, page, pageSize, totalCount int) {
	totalPages := totalCount / pageSize
	if totalCount%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Success: true,
		Data:    data,
		Message: message,
		Pagination: Pagination{
			Page:       page,
			TotalCount: totalCount,
			TotalPages: totalPages,
		},
	})
}
