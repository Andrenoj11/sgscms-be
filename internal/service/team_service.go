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
	ErrTeamNameExists = errors.New(
		"team name already exists",
	)

	ErrTeamSlugExists = errors.New(
		"team slug already exists",
	)

	ErrInvalidTeamName = errors.New(
		"invalid team name",
	)

	ErrInvalidTeamPosition = errors.New(
		"invalid team position",
	)

	ErrInvalidTeamPracticeArea = errors.New(
		"invalid team practice area",
	)
)

type TeamService struct {
	teamRepository         repository.TeamRepository
	practiceAreaRepository repository.PracticeAreaRepository
}

func NewTeamService(
	teamRepository repository.TeamRepository,
	practiceAreaRepository repository.PracticeAreaRepository,
) *TeamService {
	return &TeamService{
		teamRepository:         teamRepository,
		practiceAreaRepository: practiceAreaRepository,
	}
}

func (s *TeamService) Create(
	ctx context.Context,
	request dto.CreateTeamRequest,
	adminID string,
) (*dto.TeamResponse, error) {
	name := strings.TrimSpace(request.Name)
	position := strings.TrimSpace(request.Position)

	if name == "" {
		return nil, ErrInvalidTeamName
	}

	if position == "" {
		return nil, ErrInvalidTeamPosition
	}

	slugValue := helper.GenerateSlug(name)
	if slugValue == "" {
		return nil, ErrInvalidTeamName
	}

	if err := s.validateDuplicate(
		ctx,
		"",
		name,
		slugValue,
	); err != nil {
		return nil, err
	}

	practiceAreaIDs, err :=
		s.validatePracticeAreaIDs(
			ctx,
			request.PracticeAreaIDs,
		)
	if err != nil {
		return nil, err
	}

	createdBy := adminID

	team := &domain.Team{
		Name:             name,
		Slug:             slugValue,
		Degree:           strings.TrimSpace(request.Degree),
		Position:         position,
		ShortDescription: strings.TrimSpace(request.ShortDescription),
		Biography:        strings.TrimSpace(request.Biography),
		PhotoURL:         strings.TrimSpace(request.PhotoURL),
		Email: strings.ToLower(
			strings.TrimSpace(request.Email),
		),
		LinkedInURL: strings.TrimSpace(
			request.LinkedInURL,
		),
		DisplayOrder: request.DisplayOrder,
		IsPublished:  request.IsPublished,
		CreatedBy:    &createdBy,
		UpdatedBy:    &createdBy,
	}

	if err := s.teamRepository.Create(
		ctx,
		team,
		practiceAreaIDs,
	); err != nil {
		return nil, fmt.Errorf(
			"create team: %w",
			err,
		)
	}

	createdTeam, err := s.teamRepository.FindByID(
		ctx,
		team.ID,
	)
	if err != nil {
		return nil, err
	}

	response := mapTeamResponse(createdTeam)

	return &response, nil
}

func (s *TeamService) GetByID(
	ctx context.Context,
	id string,
) (*dto.TeamResponse, error) {
	team, err := s.teamRepository.FindByID(
		ctx,
		id,
	)
	if err != nil {
		return nil, err
	}

	response := mapTeamResponse(team)

	return &response, nil
}

func (s *TeamService) List(
	ctx context.Context,
	filter dto.TeamListQuery,
) (
	[]dto.TeamResponse,
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

	teams, total, err := s.teamRepository.List(
		ctx,
		filter,
	)
	if err != nil {
		return nil, dto.PaginationMeta{}, err
	}

	responses := make(
		[]dto.TeamResponse,
		0,
		len(teams),
	)

	for index := range teams {
		responses = append(
			responses,
			mapTeamResponse(&teams[index]),
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

func (s *TeamService) Update(
	ctx context.Context,
	id string,
	request dto.UpdateTeamRequest,
	adminID string,
) (*dto.TeamResponse, error) {
	team, err := s.teamRepository.FindByID(
		ctx,
		id,
	)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(request.Name)
	position := strings.TrimSpace(request.Position)

	if name == "" {
		return nil, ErrInvalidTeamName
	}

	if position == "" {
		return nil, ErrInvalidTeamPosition
	}

	slugValue := helper.GenerateSlug(name)
	if slugValue == "" {
		return nil, ErrInvalidTeamName
	}

	if err := s.validateDuplicate(
		ctx,
		id,
		name,
		slugValue,
	); err != nil {
		return nil, err
	}

	practiceAreaIDs, err :=
		s.validatePracticeAreaIDs(
			ctx,
			request.PracticeAreaIDs,
		)
	if err != nil {
		return nil, err
	}

	updatedBy := adminID

	team.Name = name
	team.Slug = slugValue
	team.Degree = strings.TrimSpace(request.Degree)
	team.Position = position
	team.ShortDescription = strings.TrimSpace(
		request.ShortDescription,
	)
	team.Biography = strings.TrimSpace(
		request.Biography,
	)
	team.PhotoURL = strings.TrimSpace(
		request.PhotoURL,
	)
	team.Email = strings.ToLower(
		strings.TrimSpace(request.Email),
	)
	team.LinkedInURL = strings.TrimSpace(
		request.LinkedInURL,
	)
	team.DisplayOrder = request.DisplayOrder
	team.IsPublished = request.IsPublished
	team.UpdatedBy = &updatedBy

	if err := s.teamRepository.Update(
		ctx,
		team,
		practiceAreaIDs,
	); err != nil {
		return nil, err
	}

	updatedTeam, err := s.teamRepository.FindByID(
		ctx,
		id,
	)
	if err != nil {
		return nil, err
	}

	response := mapTeamResponse(updatedTeam)

	return &response, nil
}

func (s *TeamService) Delete(
	ctx context.Context,
	id string,
	adminID string,
) error {
	_, err := s.teamRepository.FindByID(
		ctx,
		id,
	)
	if err != nil {
		return err
	}

	return s.teamRepository.SoftDelete(
		ctx,
		id,
		adminID,
	)
}

func (s *TeamService) validateDuplicate(
	ctx context.Context,
	currentID string,
	name string,
	slugValue string,
) error {
	existingByName, err :=
		s.teamRepository.FindByName(
			ctx,
			name,
		)

	if err == nil &&
		existingByName != nil &&
		existingByName.ID != currentID {
		return ErrTeamNameExists
	}

	if err != nil &&
		!errors.Is(err, repository.ErrTeamNotFound) {
		return fmt.Errorf(
			"check team name: %w",
			err,
		)
	}

	existingBySlug, err :=
		s.teamRepository.FindBySlug(
			ctx,
			slugValue,
		)

	if err == nil &&
		existingBySlug != nil &&
		existingBySlug.ID != currentID {
		return ErrTeamSlugExists
	}

	if err != nil &&
		!errors.Is(err, repository.ErrTeamNotFound) {
		return fmt.Errorf(
			"check team slug: %w",
			err,
		)
	}

	return nil
}

func (s *TeamService) validatePracticeAreaIDs(
	ctx context.Context,
	practiceAreaIDs []string,
) ([]string, error) {
	uniqueIDs := make(map[string]struct{})
	validatedIDs := make([]string, 0)

	for _, practiceAreaID := range practiceAreaIDs {
		practiceAreaID = strings.TrimSpace(
			practiceAreaID,
		)

		if !helper.IsValidUUID(practiceAreaID) {
			return nil, ErrInvalidTeamPracticeArea
		}

		if _, exists := uniqueIDs[practiceAreaID]; exists {
			continue
		}

		practiceArea, err :=
			s.practiceAreaRepository.FindByID(
				ctx,
				practiceAreaID,
			)

		if errors.Is(
			err,
			repository.ErrPracticeAreaNotFound,
		) {
			return nil, ErrInvalidTeamPracticeArea
		}

		if err != nil {
			return nil, fmt.Errorf(
				"validate team practice area: %w",
				err,
			)
		}

		if !practiceArea.IsActive {
			return nil, ErrInvalidTeamPracticeArea
		}

		uniqueIDs[practiceAreaID] = struct{}{}

		validatedIDs = append(
			validatedIDs,
			practiceAreaID,
		)
	}

	return validatedIDs, nil
}

func mapTeamResponse(
	team *domain.Team,
) dto.TeamResponse {
	practiceAreas := make(
		[]dto.PracticeAreaResponse,
		0,
		len(team.PracticeAreas),
	)

	for index := range team.PracticeAreas {
		practiceArea := team.PracticeAreas[index]

		practiceAreas = append(
			practiceAreas,
			dto.PracticeAreaResponse{
				ID:           practiceArea.ID,
				Name:         practiceArea.Name,
				Slug:         practiceArea.Slug,
				Description:  practiceArea.Description,
				IsActive:     practiceArea.IsActive,
				DisplayOrder: practiceArea.DisplayOrder,
				CreatedAt:    practiceArea.CreatedAt,
				UpdatedAt:    practiceArea.UpdatedAt,
			},
		)
	}

	return dto.TeamResponse{
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
		IsPublished:      team.IsPublished,
		PublishedAt:      team.PublishedAt,
		PracticeAreas:    practiceAreas,
		CreatedAt:        team.CreatedAt,
		UpdatedAt:        team.UpdatedAt,
	}
}