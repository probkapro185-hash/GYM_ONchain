package domain

import "time"

type TrainerSpecialization string

const (
	SpecBodyRelief TrainerSpecialization = "body_relief"
	SpecWeightLoss TrainerSpecialization = "weight_loss"
	SpecMassGain   TrainerSpecialization = "mass_gain"
)

type Trainer struct {
	ID              int64                 `json:"id"`
	UserID          int64                 `json:"user_id"`
	FullName        string                `json:"full_name"`
	Specialization  TrainerSpecialization `json:"specialization"`
	Bio             string                `json:"bio"`
	PhotoURL        string                `json:"photo_url"`
	ExperienceYears int                   `json:"experience_years"`
	IsActive        bool                  `json:"is_active"`
	CreatedAt       time.Time             `json:"created_at"`
}

type CreateTrainerInput struct {
	UserID          int64                 `json:"user_id"`
	Specialization  TrainerSpecialization `json:"specialization"`
	Bio             string                `json:"bio"`
	PhotoURL        string                `json:"photo_url"`
	ExperienceYears int                   `json:"experience_years"`
}

type UpdateTrainerInput struct {
	Specialization  TrainerSpecialization `json:"specialization"`
	Bio             string                `json:"bio"`
	PhotoURL        string                `json:"photo_url"`
	ExperienceYears int                   `json:"experience_years"`
	IsActive        bool                  `json:"is_active"`
}
