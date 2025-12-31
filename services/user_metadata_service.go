package services

import (
	"context"
	"fmt"
	"questionarie-service/models"
	"questionarie-service/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserMetadataService handles business logic for user metadata
type UserMetadataService struct {
	userMetadataRepo *repository.UserMetadataRepository
	companyRepo      *repository.CompanyRepository
}

// NewUserMetadataService creates a new UserMetadataService
func NewUserMetadataService(
	userMetadataRepo *repository.UserMetadataRepository,
	companyRepo *repository.CompanyRepository,
) *UserMetadataService {
	return &UserMetadataService{
		userMetadataRepo: userMetadataRepo,
		companyRepo:      companyRepo,
	}
}

// CreateUserMetadata creates user metadata (Super Admin only)
func (s *UserMetadataService) CreateUserMetadata(ctx context.Context, userID string, companyID primitive.ObjectID, supervisorID, department string) (*models.UserMetadata, error) {
	// Validate user ID
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// Check if metadata already exists
	exists, err := s.userMetadataRepo.Exists(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check metadata existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("user metadata already exists")
	}

	// Validate company exists
	if _, err := s.companyRepo.GetByID(ctx, companyID); err != nil {
		return nil, fmt.Errorf("company not found: %w", err)
	}

	// Validate supervisor exists if provided
	if supervisorID != "" {
		supervisorExists, err := s.userMetadataRepo.Exists(ctx, supervisorID)
		if err != nil {
			return nil, fmt.Errorf("failed to check supervisor existence: %w", err)
		}
		if !supervisorExists {
			return nil, fmt.Errorf("supervisor not found")
		}
	}

	// Create metadata
	metadata := models.NewUserMetadata(userID, companyID)
	if supervisorID != "" {
		metadata.SetSupervisor(supervisorID)
	}
	if department != "" {
		metadata.SetDepartment(department)
	}

	if err := s.userMetadataRepo.Create(ctx, metadata); err != nil {
		return nil, fmt.Errorf("failed to create user metadata: %w", err)
	}

	return metadata, nil
}

// GetUserMetadata retrieves user metadata by user ID
func (s *UserMetadataService) GetUserMetadata(ctx context.Context, userID string) (*models.UserMetadata, error) {
	return s.userMetadataRepo.GetByID(ctx, userID)
}

// GetUsersByCompany retrieves all users for a company
func (s *UserMetadataService) GetUsersByCompany(ctx context.Context, companyID primitive.ObjectID, page, pageSize int64) ([]*models.UserMetadata, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	return s.userMetadataRepo.GetUsersByCompanyWithPagination(ctx, companyID, page, pageSize)
}

// GetUsersBySupervisor retrieves users supervised by a supervisor
func (s *UserMetadataService) GetUsersBySupervisor(ctx context.Context, supervisorID string) ([]*models.UserMetadata, error) {
	return s.userMetadataRepo.GetBySupervisorID(ctx, supervisorID)
}

// UpdateUserMetadata updates user metadata
func (s *UserMetadataService) UpdateUserMetadata(ctx context.Context, userID string, companyID primitive.ObjectID, supervisorID, department string) error {
	// Get existing metadata
	metadata, err := s.userMetadataRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// Validate company if being changed
	if !companyID.IsZero() && companyID != metadata.CompanyID {
		if _, err := s.companyRepo.GetByID(ctx, companyID); err != nil {
			return fmt.Errorf("company not found: %w", err)
		}
		metadata.CompanyID = companyID
	}

	// Validate supervisor if being changed
	if supervisorID != "" && supervisorID != metadata.SupervisorID {
		supervisorExists, err := s.userMetadataRepo.Exists(ctx, supervisorID)
		if err != nil {
			return fmt.Errorf("failed to check supervisor existence: %w", err)
		}
		if !supervisorExists {
			return fmt.Errorf("supervisor not found")
		}
		metadata.SetSupervisor(supervisorID)
	}

	// Update department if provided
	if department != "" {
		metadata.SetDepartment(department)
	}

	return s.userMetadataRepo.Update(ctx, userID, metadata)
}

// DeleteUserMetadata deletes user metadata (Super Admin only)
func (s *UserMetadataService) DeleteUserMetadata(ctx context.Context, userID string) error {
	// Check if user has supervised users
	supervisedUsers, err := s.userMetadataRepo.GetBySupervisorID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to check supervised users: %w", err)
	}

	if len(supervisedUsers) > 0 {
		return fmt.Errorf("cannot delete user who supervises other users")
	}

	return s.userMetadataRepo.Delete(ctx, userID)
}

// AssignSupervisor assigns or updates a supervisor for a user
func (s *UserMetadataService) AssignSupervisor(ctx context.Context, userID, supervisorID string) error {
	// Validate supervisor exists
	if supervisorID != "" {
		supervisorExists, err := s.userMetadataRepo.Exists(ctx, supervisorID)
		if err != nil {
			return fmt.Errorf("failed to check supervisor existence: %w", err)
		}
		if !supervisorExists {
			return fmt.Errorf("supervisor not found")
		}

		// Prevent self-supervision
		if userID == supervisorID {
			return fmt.Errorf("user cannot supervise themselves")
		}
	}

	return s.userMetadataRepo.UpdateSupervisor(ctx, userID, supervisorID)
}

// GetCompanyDepartments retrieves all departments for a company
func (s *UserMetadataService) GetCompanyDepartments(ctx context.Context, companyID primitive.ObjectID) ([]string, error) {
	return s.userMetadataRepo.GetDepartmentsByCompany(ctx, companyID)
}

// GetUsersByDepartment retrieves users in a specific department
func (s *UserMetadataService) GetUsersByDepartment(ctx context.Context, companyID primitive.ObjectID, department string) ([]*models.UserMetadata, error) {
	if department == "" {
		return nil, fmt.Errorf("department is required")
	}

	return s.userMetadataRepo.GetByCompanyAndDepartment(ctx, companyID, department)
}
