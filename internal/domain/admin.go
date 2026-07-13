package domain

import "time"

type AdminRole string

const (
	AdminRoleSuperAdmin AdminRole = "super_admin"
	AdminRoleAdmin      AdminRole = "admin"
	AdminRoleEditor     AdminRole = "editor"
)

type Admin struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	Role         AdminRole
	IsActive     bool
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}