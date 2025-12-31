package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Company represents a company entity
type Company struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name      string             `bson:"name" json:"name" validate:"required,min=3,max=200"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// NewCompany creates a new Company with timestamps
func NewCompany(name string) *Company {
	now := time.Now()
	return &Company{
		ID:        primitive.NewObjectID(),
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
