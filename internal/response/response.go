package response

import (
	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Success bool `json:"success"`
	Message string `json:"message"`
	Data    any `json:"data,omitempty"`
	Meta    any `json:"meta,omitempty"`
	Errors  any `json:"errors,omitempty"`
}

func Success(
	c *gin.Context,
	statusCode int,
	message string,
	data any,
) {
	c.JSON(statusCode, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func SuccessWithMeta(
	c *gin.Context,
	statusCode int,
	message string,
	data any,
	meta any,
) {
	c.JSON(statusCode, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

func Error(
	c *gin.Context,
	statusCode int,
	message string,
	errors any,
) {
	c.JSON(statusCode, APIResponse{
		Success: false,
		Message: message,
		Errors:  errors,
	})
}