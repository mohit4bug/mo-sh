package models

import "time"

type Source struct {
	ID           string    `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Type         string    `json:"type" db:"type"`
	HasGithubApp bool      `json:"hasGithubApp" db:"has_github_app"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at"`
}

type CreateSource struct {
	Name string `json:"name" binding:"required"`
	Type string `json:"type" binding:"required"`
}
