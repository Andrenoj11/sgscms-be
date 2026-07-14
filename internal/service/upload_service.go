package service

import (
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Andrenoj11/sgscms-be/internal/config"
	"github.com/Andrenoj11/sgscms-be/internal/domain"
	"github.com/Andrenoj11/sgscms-be/internal/dto"
	"github.com/google/uuid"
	_ "golang.org/x/image/webp"
)

var (
	ErrUploadFileRequired = errors.New(
		"upload file is required",
	)

	ErrInvalidUploadCategory = errors.New(
		"invalid upload category",
	)

	ErrUploadFileTooLarge = errors.New(
		"upload file is too large",
	)

	ErrUnsupportedImageType = errors.New(
		"unsupported image type",
	)

	ErrInvalidImageFile = errors.New(
		"invalid image file",
	)
)

type UploadService struct {
	directory    string
	baseURL      string
	maxImageSize int64
}

func NewUploadService(
	cfg config.StorageConfig,
) *UploadService {
	return &UploadService{
		directory: strings.TrimSpace(
			cfg.Directory,
		),
		baseURL: strings.TrimRight(
			cfg.BaseURL,
			"/",
		),
		maxImageSize: cfg.MaxImageSize,
	}
}

func (s *UploadService) MaxImageSize() int64 {
	return s.maxImageSize
}

func (s *UploadService) UploadImage(
	fileHeader *multipart.FileHeader,
	category domain.UploadCategory,
) (*dto.UploadImageResponse, error) {
	if fileHeader == nil {
		return nil, ErrUploadFileRequired
	}

	if !category.IsValid() {
		return nil, ErrInvalidUploadCategory
	}

	if fileHeader.Size <= 0 {
		return nil, ErrInvalidImageFile
	}

	if fileHeader.Size > s.maxImageSize {
		return nil, ErrUploadFileTooLarge
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf(
			"open uploaded image: %w",
			err,
		)
	}
	defer file.Close()

	contentType, extension, err :=
		detectImageType(file)
	if err != nil {
		return nil, err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf(
			"reset uploaded image before validation: %w",
			err,
		)
	}

	imageConfig, _, err := image.DecodeConfig(file)
	if err != nil {
		return nil, ErrInvalidImageFile
	}

	if imageConfig.Width <= 0 ||
		imageConfig.Height <= 0 {
		return nil, ErrInvalidImageFile
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf(
			"reset uploaded image before saving: %w",
			err,
		)
	}

	filename := uuid.NewString() + extension

	relativeDirectory := filepath.Join(
		"images",
		string(category),
	)

	absoluteDirectory := filepath.Join(
		s.directory,
		relativeDirectory,
	)

	if err := os.MkdirAll(
		absoluteDirectory,
		0o755,
	); err != nil {
		return nil, fmt.Errorf(
			"create upload directory: %w",
			err,
		)
	}

	relativePath := filepath.Join(
		relativeDirectory,
		filename,
	)

	absolutePath := filepath.Join(
		s.directory,
		relativePath,
	)

	destination, err := os.OpenFile(
		absolutePath,
		os.O_WRONLY|
			os.O_CREATE|
			os.O_EXCL,
		0o644,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"create uploaded image file: %w",
			err,
		)
	}

	saveSucceeded := false

	defer func() {
		_ = destination.Close()

		if !saveSucceeded {
			_ = os.Remove(absolutePath)
		}
	}()

	written, err := io.Copy(
		destination,
		io.LimitReader(
			file,
			s.maxImageSize+1,
		),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"save uploaded image: %w",
			err,
		)
	}

	if written > s.maxImageSize {
		return nil, ErrUploadFileTooLarge
	}

	if err := destination.Sync(); err != nil {
		return nil, fmt.Errorf(
			"sync uploaded image: %w",
			err,
		)
	}

	if err := destination.Close(); err != nil {
		return nil, fmt.Errorf(
			"close uploaded image: %w",
			err,
		)
	}

	saveSucceeded = true

	publicPath := filepath.ToSlash(
		relativePath,
	)

	fileURL := s.baseURL + "/" + publicPath

	response := &dto.UploadImageResponse{
		Filename:     filename,
		OriginalName: filepath.Base(fileHeader.Filename),
		URL:          fileURL,
		Path:         publicPath,
		ContentType:  contentType,
		Size:         written,
		Width:        imageConfig.Width,
		Height:       imageConfig.Height,
		Category:     string(category),
	}

	return response, nil
}

func detectImageType(
	file multipart.File,
) (
	contentType string,
	extension string,
	err error,
) {
	header := make([]byte, 512)

	bytesRead, readErr := file.Read(header)

	if readErr != nil &&
		!errors.Is(readErr, io.EOF) {
		return "", "", fmt.Errorf(
			"read uploaded image header: %w",
			readErr,
		)
	}

	if bytesRead == 0 {
		return "", "", ErrInvalidImageFile
	}

	contentType = http.DetectContentType(
		header[:bytesRead],
	)

	switch contentType {
	case "image/jpeg":
		return contentType, ".jpg", nil

	case "image/png":
		return contentType, ".png", nil

	case "image/webp":
		return contentType, ".webp", nil

	default:
		return "", "", ErrUnsupportedImageType
	}
}