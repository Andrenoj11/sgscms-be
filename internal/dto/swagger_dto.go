package dto

type SwaggerErrorResponse struct {
	Success bool `json:"success" example:"false"`

	Message string `json:"message" example:"Invalid request"`

	Errors any `json:"errors,omitempty"`
}

type SwaggerSuccessResponse struct {
	Success bool `json:"success" example:"true"`

	Message string `json:"message" example:"Request successful"`

	Data any `json:"data,omitempty"`
}

type SwaggerPaginationMeta struct {
	Page int `json:"page" example:"1"`

	Limit int `json:"limit" example:"10"`

	Total int64 `json:"total" example:"25"`

	TotalPages int `json:"total_pages" example:"3"`
}

type SwaggerPaginatedResponse struct {
	Success bool `json:"success" example:"true"`

	Message string `json:"message" example:"Data retrieved successfully"`

	Data any `json:"data"`

	Meta SwaggerPaginationMeta `json:"meta"`
}