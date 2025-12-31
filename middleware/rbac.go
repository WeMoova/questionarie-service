package middleware

import (
	"context"
	"net/http"
)

// Role constants
const (
	RoleSuperAdmin   = "super_admin"
	RoleCompanyAdmin = "company_admin"
	RoleSupervisor   = "supervisor"
	RoleEmployee     = "employee"
)

// RequireRole creates a middleware that checks if the user has one of the required roles
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := GetUserFromContext(r.Context())
			if err != nil || claims == nil {
				http.Error(w, "Unauthorized: no user claims found", http.StatusUnauthorized)
				return
			}

			// Check if user has any of the required roles
			hasRole := false
			for _, requiredRole := range roles {
				for _, userRole := range claims.Roles {
					if userRole == requiredRole {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}

			if !hasRole {
				http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireSuperAdmin creates a middleware that requires super admin role
func RequireSuperAdmin() func(http.Handler) http.Handler {
	return RequireRole(RoleSuperAdmin)
}

// RequireCompanyAdmin creates a middleware that requires company admin or super admin role
func RequireCompanyAdmin() func(http.Handler) http.Handler {
	return RequireRole(RoleSuperAdmin, RoleCompanyAdmin)
}

// RequireSupervisor creates a middleware that requires supervisor, company admin, or super admin role
func RequireSupervisor() func(http.Handler) http.Handler {
	return RequireRole(RoleSuperAdmin, RoleCompanyAdmin, RoleSupervisor)
}

// RequireEmployee creates a middleware that allows any authenticated user (all roles)
func RequireEmployee() func(http.Handler) http.Handler {
	return RequireRole(RoleSuperAdmin, RoleCompanyAdmin, RoleSupervisor, RoleEmployee)
}

// HasRole checks if the user has a specific role
func HasRole(ctx context.Context, role string) bool {
	claims, err := GetUserFromContext(ctx)
	if err != nil || claims == nil {
		return false
	}

	for _, userRole := range claims.Roles {
		if userRole == role {
			return true
		}
	}
	return false
}

// HasAnyRole checks if the user has any of the specified roles
func HasAnyRole(ctx context.Context, roles ...string) bool {
	claims, err := GetUserFromContext(ctx)
	if err != nil || claims == nil {
		return false
	}

	for _, requiredRole := range roles {
		for _, userRole := range claims.Roles {
			if userRole == requiredRole {
				return true
			}
		}
	}
	return false
}

// IsSuperAdmin checks if the user is a super admin
func IsSuperAdmin(ctx context.Context) bool {
	return HasRole(ctx, RoleSuperAdmin)
}

// IsCompanyAdmin checks if the user is a company admin or super admin
func IsCompanyAdmin(ctx context.Context) bool {
	return HasAnyRole(ctx, RoleSuperAdmin, RoleCompanyAdmin)
}

// IsSupervisor checks if the user is a supervisor, company admin, or super admin
func IsSupervisor(ctx context.Context) bool {
	return HasAnyRole(ctx, RoleSuperAdmin, RoleCompanyAdmin, RoleSupervisor)
}

// GetUserID retrieves the user ID from the context
func GetUserID(ctx context.Context) string {
	claims, err := GetUserFromContext(ctx)
	if err != nil || claims == nil {
		return ""
	}
	return claims.Sub
}

// GetUserEmail retrieves the user email from the context
func GetUserEmail(ctx context.Context) string {
	claims, err := GetUserFromContext(ctx)
	if err != nil || claims == nil {
		return ""
	}
	return claims.Email
}
