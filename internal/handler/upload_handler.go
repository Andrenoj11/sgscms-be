package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Andrenoj11/sgscms-be/internal/domain"
	"github.com/Andrenoj11/sgscms-be/internal/response"
	"github.com/Andrenoj11/sgscms-be/internal/service"
	"github.com/gin-gonic/gin"
)

const multipartAdditionalMemory int64 = 1 * 1024 * 1024

type UploadHandler struct {
	service *service.UploadService
}

func NewUploadHandler(
	uploadService *service.UploadService,
) *UploadHandler {
	return &UploadHandler{
		service: uploadService,
	}
}

func (h *UploadHandler) UploadImage(
	c *gin.Context,
) {
	maxRequestSize :=
		h.service.MaxImageSize() +
			multipartAdditionalMemory

	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		maxRequestSize,
	)

	if err := c.Request.ParseMultipartForm(
		maxRequestSize,
	); err != nil {
		response.Error(
			c,
			http.StatusRequestEntityTooLarge,
			"Uploaded image is too large",
			nil,
		)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.Error(
			c,
			http.StatusBadRequest,
			"Image file is required",
			nil,
		)
		return
	}

	category := domain.UploadCategory(
		strings.ToLower(
			strings.TrimSpace(
				c.PostForm("category"),
			),
		),
	)

	uploadedImage, err :=
		h.service.UploadImage(
			fileHeader,
			category,
		)

	switch {
	case errors.Is(
		err,
		service.ErrUploadFileRequired,
	):
		response.Error(
			c,
			http.StatusBadRequest,
			"Image file is required",
			nil,
		)
		return

	case errors.Is(
		err,
		service.ErrInvalidUploadCategory,
	):
		response.Error(
			c,
			http.StatusBadRequest,
			"Image category must be team or news",
			nil,
		)
		return

	case errors.Is(
		err,
		service.ErrUploadFileTooLarge,
	):
		response.Error(
			c,
			http.StatusRequestEntityTooLarge,
			"Uploaded image is too large",
			nil,
		)
		return

	case errors.Is(
		err,
		service.ErrUnsupportedImageType,
	):
		response.Error(
			c,
			http.StatusUnsupportedMediaType,
			"Image type must be JPEG, PNG, or WEBP",
			nil,
		)
		return

	case errors.Is(
		err,
		service.ErrInvalidImageFile,
	):
		response.Error(
			c,
			http.StatusBadRequest,
			"Uploaded file is not a valid image",
			nil,
		)
		return

	case err != nil:
		response.Error(
			c,
			http.StatusInternalServerError,
			"Internal server error",
			nil,
		)
		return
	}

	response.Success(
		c,
		http.StatusCreated,
		"Image uploaded successfully",
		uploadedImage,
	)
}