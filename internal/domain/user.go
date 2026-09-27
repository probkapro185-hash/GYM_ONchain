package domain

import "time"

type Role string

const (
	RoleClient  Role = "client"
	RoleManager Role = "manager"
	RoleAdmin   Role = "admin"
)

type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
)

type User struct {
	ID                    int64      `json:"id"`
	FullName              string     `json:"full_name"`
	Phone                 string     `json:"phone"`
	Email                 string     `json:"email"`
	PasswordHash          string     `json:"-"`
	TokenVersion          int        `json:"-"`
	Role                  Role       `json:"role"`
	Gender                Gender     `json:"gender"`
	Balance               float64    `json:"balance"`
	Visits                int        `json:"visits"`
	IsActive              bool       `json:"is_active"`
	PasswordSetupRequired bool       `json:"password_setup_required"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
	LastVisitAt           *time.Time `json:"last_visit_at,omitempty"`
	SessionsLeft          *int       `json:"sessions_left,omitempty"`
}

type ApplicationStatus string

const (
	ApplicationPending  ApplicationStatus = "pending"
	ApplicationApproved ApplicationStatus = "approved"
	ApplicationRejected ApplicationStatus = "rejected"
)

type ApplicationRequest struct {
	Gender    Gender            `json:"gender"`
	ID        int64             `json:"id"`
	FullName  string            `json:"full_name"`
	Phone     string            `json:"phone"`
	Email     string            `json:"email"`
	Status    ApplicationStatus `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
}

type CreateApplicationInput struct {
	Gender   Gender `json:"gender"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateUserInput struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Gender   Gender `json:"gender"`
}

type ChangePasswordInput struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type CreateUserInput struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     Role   `json:"role"`
	Gender   Gender `json:"gender"`
}
