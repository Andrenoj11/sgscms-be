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
	ErrPracticeAreaNameExists = errors.New(
		"practice area name already exists",
	)

	ErrPracticeAreaSlugExists = errors.New(
		"practice area slug already exists",
	)

	ErrInvalidPracticeAreaName = errors.New(
		"invalid practice area name",
	)
)

type PracticeAreaService struct {
	repository repository.PracticeAreaRepository
}

func NewPracticeAreaService(
	practiceAreaRepository repository.PracticeAreaRepository,
) *PracticeAreaService {
	return &PracticeAreaService{
		repository: practiceAreaRepository,
	}
}

func (s *PracticeAreaService) Create(
	ctx context.Context,
	request dto.CreatePracticeAreaRequest,
	adminID string,
) (*dto.PracticeAreaResponse, error) {
	name := strings.TrimSpace(request.Name)

	if name == "" {
		return nil, ErrInvalidPracticeAreaName
	}

	slugValue := helper.GenerateSlug(name)
	if slugValue == "" {
		return nil, ErrInvalidPracticeAreaName
	}

	existingByName, err := s.repository.FindByName(
		ctx,
		name,
	)
	if err == nil && existingByName != nil {
		return nil, ErrPracticeAreaNameExists
	}

	if !errors.Is(
		err,
		repository.ErrPracticeAreaNotFound,
	) {
		return nil, fmt.Errorf(
			"check practice area name: %w",
			err,
		)
	}

	existingBySlug, err := s.repository.FindBySlug(
		ctx,
		slugValue,
	)
	if err == nil && existingBySlug != nil {
		return nil, ErrPracticeAreaSlugExists
	}

	if !errors.Is(
		err,
		repository.ErrPracticeAreaNotFound,
	) {
		return nil, fmt.Errorf(
			"check practice area slug: %w",
			err,
		)
	}

	isActive := true

	if request.IsActive != nil {
		isActive = *request.IsActive
	}

	createdBy := adminID

	practiceArea := &domain.PracticeArea{
		Name:         name,
		Slug:         slugValue,
		Description:  strings.TrimSpace(request.Description),
		IsActive:     isActive,
		DisplayOrder: request.DisplayOrder,
		CreatedBy:    &createdBy,
		UpdatedBy:    &createdBy,
	}

	if err := s.repository.Create(
		ctx,
		practiceArea,
	); err != nil {
		return nil, fmt.Errorf(
			"create practice area: %w",
			err,
		)
	}

	response := mapPracticeAreaResponse(
		practiceArea,
	)

	return &response, nil
}

func (s *PracticeAreaService) GetByID(
	ctx context.Context,
	id string,
) (*dto.PracticeAreaResponse, error) {
	practiceArea, err := s.repository.FindByID(
		ctx,
		id,
	)
	if err != nil {
		return nil, err
	}

	response := mapPracticeAreaResponse(
		practiceArea,
	)

	return &response, nil
}

func (s *PracticeAreaService) List(
	ctx context.Context,
	query dto.PracticeAreaListQuery,
) (
	[]dto.PracticeAreaResponse,
	dto.PaginationMeta,
	error,
) {
	if query.Page < 1 {
		query.Page = 1
	}

	if query.Limit < 1 {
		query.Limit = 10
	}

	if query.Limit > 100 {
		query.Limit = 100
	}

	query.Search = strings.TrimSpace(query.Search)

	practiceAreas, total, err := s.repository.List(
		ctx,
		query,
	)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	responses := make(
		[]dto.PracticeAreaResponse,
		0,
		len(practiceAreas),
	)

	for index := range practiceAreas {
		responses = append(
			responses,
			mapPracticeAreaResponse(
				&practiceAreas[index],
			),
		)
	}

	totalPages := 0

	if total > 0 {
		totalPages = int(
			math.Ceil(
				float64(total) /
					float64(query.Limit),
			),
		)
	}

	meta := dto.PaginationMeta{
		Page:       query.Page,
		Limit:      query.Limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return responses, meta, nil
}

func (s *PracticeAreaService) Update(
	ctx context.Context,
	id string,
	request dto.UpdatePracticeAreaRequest,
	adminID string,
) (*dto.PracticeAreaResponse, error) {
	practiceArea, err := s.repository.FindByID(
		ctx,
		id,
	)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(request.Name)
	if name == "" {
		return nil, ErrInvalidPracticeAreaName
	}

	slugValue := helper.GenerateSlug(name)
	if slugValue == "" {
		return nil, ErrInvalidPracticeAreaName
	}

	existingByName, err := s.repository.FindByName(
		ctx,
		name,
	)

	if err == nil &&
		existingByName != nil &&
		existingByName.ID != id {
		return nil, ErrPracticeAreaNameExists
	}

	if err != nil &&
		!errors.Is(
			err,
			repository.ErrPracticeAreaNotFound,
		) {
		return nil, fmt.Errorf(
			"check practice area name: %w",
			err,
		)
	}

	existingBySlug, err := s.repository.FindBySlug(
		ctx,
		slugValue,
	)

	if err == nil &&
		existingBySlug != nil &&
		existingBySlug.ID != id {
		return nil, ErrPracticeAreaSlugExists
	}

	if err != nil &&
		!errors.Is(
			err,
			repository.ErrPracticeAreaNotFound,
		) {
		return nil, fmt.Errorf(
			"check practice area slug: %w",
			err,
		)
	}

	updatedBy := adminID

	practiceArea.Name = name
	practiceArea.Slug = slugValue
	practiceArea.Description = strings.TrimSpace(
		request.Description,
	)
	practiceArea.IsActive = request.IsActive
	practiceArea.DisplayOrder = request.DisplayOrder
	practiceArea.UpdatedBy = &updatedBy

	if err := s.repository.Update(
		ctx,
		practiceArea,
	); err != nil {
		return nil, err
	}

	response := mapPracticeAreaResponse(
		practiceArea,
	)

	return &response, nil
}

func (s *PracticeAreaService) Delete(
	ctx context.Context,
	id string,
	adminID string,
) error {
	_, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return err
	}

	return s.repository.SoftDelete(
		ctx,
		id,
		adminID,
	)
}

func mapPracticeAreaResponse(
	practiceArea *domain.PracticeArea,
) dto.PracticeAreaResponse {
	return dto.PracticeAreaResponse{
		ID:           practiceArea.ID,
		Name:         practiceArea.Name,
		Slug:         practiceArea.Slug,
		Description:  practiceArea.Description,
		IsActive:     practiceArea.IsActive,
		DisplayOrder: practiceArea.DisplayOrder,
		CreatedAt:    practiceArea.CreatedAt,
		UpdatedAt:    practiceArea.UpdatedAt,
	}
}