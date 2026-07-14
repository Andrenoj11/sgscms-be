package service

import (
	"context"
	"math"
	"strings"

	"github.com/Andrenoj11/sgscms-be/internal/domain"
	"github.com/Andrenoj11/sgscms-be/internal/dto"
	"github.com/Andrenoj11/sgscms-be/internal/repository"
)

type PublicService struct {
	publicRepository repository.PublicRepository
}

func NewPublicService(
	publicRepository repository.PublicRepository,
) *PublicService {
	return &PublicService{
		publicRepository: publicRepository,
	}
}

func (s *PublicService) ListPracticeAreas(
	ctx context.Context,
) ([]dto.PublicPracticeAreaResponse, error) {
	practiceAreas, err :=
		s.publicRepository.ListPracticeAreas(ctx)
	if err != nil {
		return nil, err
	}

	responses := make(
		[]dto.PublicPracticeAreaResponse,
		0,
		len(practiceAreas),
	)

	for index := range practiceAreas {
		responses = append(
			responses,
			mapPublicPracticeAreaResponse(
				&practiceAreas[index],
			),
		)
	}

	return responses, nil
}

func (s *PublicService) ListTeams(
	ctx context.Context,
	filter dto.PublicListQuery,
) (
	[]dto.PublicTeamListResponse,
	dto.PaginationMeta,
	error,
) {
	normalizePublicListQuery(&filter)

	teams, total, err :=
		s.publicRepository.ListTeams(
			ctx,
			filter,
		)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	responses := make(
		[]dto.PublicTeamListResponse,
		0,
		len(teams),
	)

	for index := range teams {
		responses = append(
			responses,
			mapPublicTeamListResponse(
				&teams[index],
			),
		)
	}

	meta := buildPaginationMeta(
		filter.Page,
		filter.Limit,
		total,
	)

	return responses, meta, nil
}

func (s *PublicService) GetTeamBySlug(
	ctx context.Context,
	slugValue string,
) (*dto.PublicTeamDetailResponse, error) {
	team, err :=
		s.publicRepository.FindTeamBySlug(
			ctx,
			strings.TrimSpace(slugValue),
		)
	if err != nil {
		return nil, err
	}

	response := mapPublicTeamDetailResponse(team)

	return &response, nil
}

func (s *PublicService) ListNews(
	ctx context.Context,
	filter dto.PublicNewsListQuery,
) (
	[]dto.PublicNewsListResponse,
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

	newsList, total, err :=
		s.publicRepository.ListNews(
			ctx,
			filter,
		)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	responses := make(
		[]dto.PublicNewsListResponse,
		0,
		len(newsList),
	)

	for index := range newsList {
		responses = append(
			responses,
			mapPublicNewsListResponse(
				&newsList[index],
			),
		)
	}

	meta := buildPaginationMeta(
		filter.Page,
		filter.Limit,
		total,
	)

	return responses, meta, nil
}

func (s *PublicService) GetNewsBySlug(
	ctx context.Context,
	slugValue string,
) (*dto.PublicNewsDetailResponse, error) {
	news, err :=
		s.publicRepository.FindNewsBySlug(
			ctx,
			strings.TrimSpace(slugValue),
		)
	if err != nil {
		return nil, err
	}

	response := mapPublicNewsDetailResponse(news)

	return &response, nil
}

func normalizePublicListQuery(
	filter *dto.PublicListQuery,
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
}

func buildPaginationMeta(
	page int,
	limit int,
	total int64,
) dto.PaginationMeta {
	totalPages := 0

	if total > 0 {
		totalPages = int(
			math.Ceil(
				float64(total) /
					float64(limit),
			),
		)
	}

	return dto.PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}

func mapPublicPracticeAreaResponse(
	practiceArea *domain.PracticeArea,
) dto.PublicPracticeAreaResponse {
	return dto.PublicPracticeAreaResponse{
		ID:           practiceArea.ID,
		Name:         practiceArea.Name,
		Slug:         practiceArea.Slug,
		Description:  practiceArea.Description,
		DisplayOrder: practiceArea.DisplayOrder,
	}
}

func mapPublicTeamListResponse(
	team *domain.Team,
) dto.PublicTeamListResponse {
	return dto.PublicTeamListResponse{
		ID:               team.ID,
		Name:             team.Name,
		Slug:             team.Slug,
		Degree:           team.Degree,
		Position:         team.Position,
		ShortDescription: team.ShortDescription,
		PhotoURL:         team.PhotoURL,
		DisplayOrder:     team.DisplayOrder,
		PracticeAreas: mapPublicPracticeAreas(
			team.PracticeAreas,
		),
	}
}

func mapPublicTeamDetailResponse(
	team *domain.Team,
) dto.PublicTeamDetailResponse {
	return dto.PublicTeamDetailResponse{
		ID:               team.ID,
		Name:             team.Name,
		Slug:             team.Slug,
		Degree:           team.Degree,
		Position:         team.Position,
		ShortDescription: team.ShortDescription,
		Biography:        team.Biography,
		PhotoURL:         team.PhotoURL,
		Email:            team.Email,
		LinkedInURL:      team.LinkedInURL,
		DisplayOrder:     team.DisplayOrder,
		PublishedAt:      team.PublishedAt,
		PracticeAreas: mapPublicPracticeAreas(
			team.PracticeAreas,
		),
	}
}

func mapPublicPracticeAreas(
	practiceAreas []domain.PracticeArea,
) []dto.PublicPracticeAreaResponse {
	responses := make(
		[]dto.PublicPracticeAreaResponse,
		0,
		len(practiceAreas),
	)

	for index := range practiceAreas {
		responses = append(
			responses,
			mapPublicPracticeAreaResponse(
				&practiceAreas[index],
			),
		)
	}

	return responses
}

func mapPublicNewsListResponse(
	news *domain.News,
) dto.PublicNewsListResponse {
	return dto.PublicNewsListResponse{
		ID:               news.ID,
		Title:            news.Title,
		Slug:             news.Slug,
		Excerpt:          news.Excerpt,
		FeaturedImageURL: news.FeaturedImageURL,
		IsFeatured:       news.IsFeatured,
		PublishedAt:      news.PublishedAt,
	}
}

func mapPublicNewsDetailResponse(
	news *domain.News,
) dto.PublicNewsDetailResponse {
	return dto.PublicNewsDetailResponse{
		ID:               news.ID,
		Title:            news.Title,
		Slug:             news.Slug,
		Excerpt:          news.Excerpt,
		Content:          news.Content,
		FeaturedImageURL: news.FeaturedImageURL,
		IsFeatured:       news.IsFeatured,
		PublishedAt:      news.PublishedAt,
	}
}