package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// APIToken represents a company API token for programmatic access
type APIToken struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	CompanyID   primitive.ObjectID `bson:"company_id" json:"company_id"`
	TokenHash   string             `bson:"token_hash" json:"-"`
	TokenPrefix string             `bson:"token_prefix" json:"token_prefix"`
	Name        string             `bson:"name" json:"name"`
	IsActive    bool               `bson:"is_active" json:"is_active"`
	LastUsedAt  *time.Time         `bson:"last_used_at,omitempty" json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time         `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	CreatedBy   string             `bson:"created_by" json:"created_by"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	RevokedAt   *time.Time         `bson:"revoked_at,omitempty" json:"revoked_at,omitempty"`
	// Transient fields — not persisted
	RawToken    string `bson:"-" json:"token,omitempty"`
	CompanyName string `bson:"-" json:"company_name,omitempty"`
}
