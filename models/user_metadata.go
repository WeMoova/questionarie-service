package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserMetadata represents the metadata that links FusionAuth users with companies
type UserMetadata struct {
	ID           string             `bson:"_id" json:"id"`                       // FusionAuth user ID (sub)
	CompanyID    primitive.ObjectID `bson:"company_id" json:"company_id" validate:"required"` // Reference to companies
	SupervisorID string             `bson:"supervisor_id,omitempty" json:"supervisor_id,omitempty"` // FusionAuth ID of supervisor
	Department   string             `bson:"department,omitempty" json:"department,omitempty"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`
}

// NewUserMetadata creates a new UserMetadata
func NewUserMetadata(userID string, companyID primitive.ObjectID) *UserMetadata {
	now := time.Now()
	return &UserMetadata{
		ID:        userID,
		CompanyID: companyID,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetSupervisor sets the supervisor for this user
func (u *UserMetadata) SetSupervisor(supervisorID string) {
	u.SupervisorID = supervisorID
	u.UpdatedAt = time.Now()
}

// SetDepartment sets the department for this user
func (u *UserMetadata) SetDepartment(department string) {
	u.Department = department
	u.UpdatedAt = time.Now()
}

// BelongsToCompany checks if the user belongs to the given company
func (u *UserMetadata) BelongsToCompany(companyID primitive.ObjectID) bool {
	return u.CompanyID == companyID
}

// HasSupervisor checks if the user has a supervisor
func (u *UserMetadata) HasSupervisor() bool {
	return u.SupervisorID != ""
}

// IsSupervisedBy checks if the user is supervised by the given supervisor ID
func (u *UserMetadata) IsSupervisedBy(supervisorID string) bool {
	return u.SupervisorID == supervisorID
}
