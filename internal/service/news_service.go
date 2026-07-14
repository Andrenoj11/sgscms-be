package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/Andrenoj11/sgscms-be/internal/domain"
	"github.com/Andrenoj11/sgscms-be/internal/dto"
	"github.com/Andrenoj11/sgscms-be/internal/helper"
	"github.com/Andrenoj11/sgscms-be/internal/repository"
)

var (
	ErrNewsSlugExists = errors.New(
		"news slug already exists",
	)

	ErrInvalidNewsTitle = errors.New(
		"invalid news title",
	)

	ErrInvalidNewsContent = errors.New(
		"invalid news content",
	)

	ErrInvalidNewsStatus = errors.New(
		"invalid news status",
	)
)

type NewsService struct {
	newsRepository repository.NewsRepository
}

func NewNewsService(
	newsRepository repository.NewsRepository,
) *NewsService {
	return &NewsService{
		newsRepository: newsRepository,
	}
}

func (s *NewsService) Create(
	ctx context.Context,
	request dto.CreateNewsRequest,
	adminID string,
) (*dto.NewsResponse, error) {
	title := strings.TrimSpace(request.Title)
	content := strings.TrimSpace(request.Content)

	if title == "" {
		return nil, ErrInvalidNewsTitle
	}

	if content == "" {
		return nil, ErrInvalidNewsContent
	}

	status, err := parseNewsStatus(request.Status)
	if err != nil {
		return nil, err
	}

	slugValue := helper.GenerateSlug(title)
	if slugValue == "" {
		return nil, ErrInvalidNewsTitle
	}

	if err := s.validateSlug(
		ctx,
		"",
		slugValue,
	); err != nil {
		return nil, err
	}

	createdBy := adminID

	news := &domain.News{
		Title:            title,
		Slug:             slugValue,
		Excerpt:          strings.TrimSpace(request.Excerpt),
		Content:          content,
		FeaturedImageURL: strings.TrimSpace(request.FeaturedImageURL),
		Status:           status,
		IsFeatured:       request.IsFeatured,
		CreatedBy:        &createdBy,
		UpdatedBy:        &createdBy,
	}

	if err := s.newsRepository.Create(
		ctx,
		news,
	); err != nil {
		return nil, fmt.Errorf(
			"create news: %w",
			err,
		)
	}

	response := mapNewsResponse(news)

	return &response, nil
}

func (s *NewsService) GetByID(
	ctx context.Context,
	id string,
) (*dto.NewsResponse, error) {
	news, err := s.newsRepository.FindByID(
		ctx,
		id,
	)
	if err != nil {
		return nil, err
	}

	response := mapNewsResponse(news)

	return &response, nil
}

func (s *NewsService) List(
	ctx context.Context,
	filter dto.NewsListQuery,
) (
	[]dto.NewsResponse,
	dto.PaginationMeta,
	error,
) {
	if filter.Page < 1 {
		filter.Page = 1
	}

	if filter.Limit < 1 {
		filter.Limit = 10
	}

	if filter.Limit > 100 {
		filter.Limit = 100
	}

	filter.Search = strings.TrimSpace(
		filter.Search,
	)

	filter.Status = strings.ToLower(
		strings.TrimSpace(filter.Status),
	)

	if filter.Status != "" {
		if _, err := parseNewsStatus(filter.Status); err != nil {
			return nil, dto.PaginationMeta{}, err
		}
	}

	newsList, total, err := s.newsRepository.List(
		ctx,
		filter,
	)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	responses := make(
		[]dto.NewsResponse,
		0,
		len(newsList),
	)

	for index := range newsList {
		responses = append(
			responses,
			mapNewsResponse(&newsList[index]),
		)
	}

	totalPages := 0

	if total > 0 {
		totalPages = int(
			math.Ceil(
				float64(total) /
					float64(filter.Limit),
			),
		)
	}

	meta := dto.PaginationMeta{
		Page:       filter.Page,
		Limit:      filter.Limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return responses, meta, nil
}

func (s *NewsService) Update(
	ctx context.Context,
	id string,
	request dto.UpdateNewsRequest,
	adminID string,
) (*dto.NewsResponse, error) {
	news, err := s.newsRepository.FindByID(
		ctx,
		id,
	)
	if err != nil {
		return nil, err
	}

	title := strings.TrimSpace(request.Title)
	content := strings.TrimSpace(request.Content)

	if title == "" {
		return nil, ErrInvalidNewsTitle
	}

	if content == "" {
		return nil, ErrInvalidNewsContent
	}

	status, err := parseNewsStatus(request.Status)
	if err != nil {
		return nil, err
	}

	slugValue := helper.GenerateSlug(title)
	if slugValue == "" {
		return nil, ErrInvalidNewsTitle
	}

	if err := s.validateSlug(
		ctx,
		id,
		slugValue,
	); err != nil {
		return nil, err
	}

	updatedBy := adminID

	news.Title = title
	news.Slug = slugValue
	news.Excerpt = strings.TrimSpace(
		request.Excerpt,
	)
	news.Content = content
	news.FeaturedImageURL = strings.TrimSpace(
		request.FeaturedImageURL,
	)
	news.Status = status
	news.IsFeatured = request.IsFeatured
	news.UpdatedBy = &updatedBy

	if err := s.newsRepository.Update(
		ctx,
		news,
	); err != nil {
		return nil, err
	}

	response := mapNewsResponse(news)

	return &response, nil
}

func (s *NewsService) Delete(
	ctx context.Context,
	id string,
	adminID string,
) error {
	_, err := s.newsRepository.FindByID(
		ctx,
		id,
	)
	if err != nil {
		return err
	}

	return s.newsRepository.SoftDelete(
		ctx,
		id,
		adminID,
	)
}

func (s *NewsService) validateSlug(
	ctx context.Context,
	currentID string,
	slugValue string,
) error {
	existingNews, err :=
		s.newsRepository.FindBySlug(
			ctx,
			slugValue,
		)

	if err == nil &&
		existingNews != nil &&
		existingNews.ID != currentID {
		return ErrNewsSlugExists
	}

	if err != nil &&
		!errors.Is(
			err,
			repository.ErrNewsNotFound,
		) {
		return fmt.Errorf(
			"check news slug: %w",
			err,
		)
	}

	return nil
}

func parseNewsStatus(
	value string,
) (domain.NewsStatus, error) {
	status := domain.NewsStatus(
		strings.ToLower(
			strings.TrimSpace(value),
		),
	)

	switch status {
	case domain.NewsStatusDraft,
		domain.NewsStatusPublished,
		domain.NewsStatusArchived:
		return status, nil

	default:
		return "", ErrInvalidNewsStatus
	}
}

func mapNewsResponse(
	news *domain.News,
) dto.NewsResponse {
	return dto.NewsResponse{
		ID:               news.ID,
		Title:            news.Title,
		Slug:             news.Slug,
		Excerpt:          news.Excerpt,
		Content:          news.Content,
		FeaturedImageURL: news.FeaturedImageURL,
		Status:           string(news.Status),
		IsFeatured:       news.IsFeatured,
		PublishedAt:      news.PublishedAt,
		CreatedAt:        news.CreatedAt,
		UpdatedAt:        news.UpdatedAt,
	}
}