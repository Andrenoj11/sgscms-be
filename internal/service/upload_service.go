package service

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	driver       string
	directory    string
	baseURL      string
	maxImageSize int64
	supabaseURL  string
	supabaseKey  string
	bucket       string
	httpClient   *http.Client
}

func NewUploadService(
	cfg config.StorageConfig,
) *UploadService {
	return &UploadService{
		driver: strings.TrimSpace(cfg.Driver),
		directory: strings.TrimSpace(
			cfg.Directory,
		),
		baseURL: strings.TrimRight(
			cfg.BaseURL,
			"/",
		),
		maxImageSize: cfg.MaxImageSize,
		supabaseURL:  strings.TrimRight(cfg.SupabaseURL, "/"),
		supabaseKey:  cfg.SupabaseKey,
		bucket:       strings.Trim(cfg.Bucket, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *UploadService) IsLocal() bool {
	return s.driver == "local"
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

	relativePath := filepath.Join(
		relativeDirectory,
		filename,
	)

	content, err := io.ReadAll(io.LimitReader(file, s.maxImageSize+1))
	if err != nil {
		return nil, fmt.Errorf(
			"read uploaded image: %w",
			err,
		)
	}

	written := int64(len(content))
	if written > s.maxImageSize {
		return nil, ErrUploadFileTooLarge
	}

	publicPath := filepath.ToSlash(
		relativePath,
	)

	if err := s.save(content, contentType, publicPath); err != nil {
		return nil, err
	}

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

func (s *UploadService) save(content []byte, contentType, publicPath string) error {
	if s.IsLocal() {
		absolutePath := filepath.Join(s.directory, filepath.FromSlash(publicPath))
		if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
			return fmt.Errorf("create upload directory: %w", err)
		}

		if err := os.WriteFile(absolutePath, content, 0o644); err != nil {
			return fmt.Errorf("save uploaded image: %w", err)
		}

		return nil
	}

	objectURL := fmt.Sprintf(
		"%s/storage/v1/object/%s/%s",
		s.supabaseURL,
		url.PathEscape(s.bucket),
		strings.Join(escapePathSegments(publicPath), "/"),
	)
	req, err := http.NewRequest(http.MethodPost, objectURL, bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("create Supabase Storage request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.supabaseKey)
	req.Header.Set("apikey", s.supabaseKey)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "false")

	res, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload image to Supabase Storage: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return fmt.Errorf("Supabase Storage returned %s: %s", res.Status, strings.TrimSpace(string(body)))
	}

	return nil
}

func escapePathSegments(path string) []string {
	segments := strings.Split(filepath.ToSlash(path), "/")
	for i := range segments {
		segments[i] = url.PathEscape(segments[i])
	}
	return segments
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
