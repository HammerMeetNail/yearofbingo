package models

import "github.com/google/uuid"

type Suggestion struct {
	ID       uuid.UUID `json:"id"`
	Category string    `json:"category"`
	Content  string    `json:"content"`
	IsActive bool      `json:"is_active"`
}

var SuggestionCategories = []string{
	"Health & Fitness",
	"Career & Learning",
	"Relationships",
	"Hobbies & Creativity",
	"Finance",
	"Travel & Adventure",
	"Personal Growth",
	"Home & Organization",
}
