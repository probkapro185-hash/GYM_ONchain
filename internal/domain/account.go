package domain

import "time"

type AccountTokenPurpose string

const (
	AccountTokenActivation    AccountTokenPurpose = "activation"
	AccountTokenPasswordReset AccountTokenPurpose = "password_reset"
)

type AccountToken struct {
	ID        int64
	UserID    int64
	Purpose   AccountTokenPurpose
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

type SetPasswordInput struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

type ForgotPasswordInput struct {
	Email string `json:"email"`
}

type InviteClientInput struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Gender   Gender `json:"gender"`
}
