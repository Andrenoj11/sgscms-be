package dto

type LoginRequest struct {
	Email string `json:"email" binding:"required,email,max=150"`

	Password string `json:"password" binding:"required,min=8,max=128"`
}

type AdminResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type LoginResponse struct {
	AccessToken string        `json:"access_token"`
	TokenType   string        `json:"token_type"`
	ExpiresIn   int64         `json:"expires_in"`
	Admin       AdminResponse `json:"admin"`
}