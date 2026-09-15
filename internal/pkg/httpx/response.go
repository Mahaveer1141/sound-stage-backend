package httpx

import (
	"net/http"
	fileattachment "sound-stage-backend/internal/file_attachment"

	"github.com/gin-gonic/gin"
)

type FileAttachmentResponse struct {
	PublicID string `json:"publicId"`
	URL      string `json:"url"`
}

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
	Page       int    `json:"page"`
	TotalCount int    `json:"totalCount"`
	TotalPages int    `json:"totalPages"`
	NextCursor string `json:"nextCursor"`
	HasMore    bool   `json:"hasMore"`
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

func CursorPaginatedSuccessResponse(c *gin.Context, message string, data any, limit int, totalCount int64, nextCursor string, hasMore bool) {
	totalPages := int(totalCount) / limit
	if int(totalCount)%limit > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Success: true,
		Data:    data,
		Message: message,
		Pagination: Pagination{
			Page:       1,
			TotalCount: int(totalCount),
			TotalPages: totalPages,
			NextCursor: nextCursor,
			HasMore:    hasMore,
		},
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

func BuildFileAttachmentResponse(attachment *fileattachment.FileAttachment) *FileAttachmentResponse {
	if attachment == nil {
		return nil
	}
	return &FileAttachmentResponse{
		PublicID: attachment.PublicID,
		URL:      attachment.URL,
	}
}
