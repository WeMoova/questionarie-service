package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Branding holds visual customization for the employee-facing app
type Branding struct {
	Logo                 string `bson:"logo,omitempty" json:"logo,omitempty"`
	LogoIcon             string `bson:"logo_icon,omitempty" json:"logo_icon,omitempty"`
	Favicon              string `bson:"favicon,omitempty" json:"favicon,omitempty"`
	PrimaryColor         string `bson:"primary_color,omitempty" json:"primary_color,omitempty"`
	SecondaryColor       string `bson:"secondary_color,omitempty" json:"secondary_color,omitempty"`
	AccentColor          string `bson:"accent_color,omitempty" json:"accent_color,omitempty"`
	LoginBackgroundImage string `bson:"login_background_image,omitempty" json:"login_background_image,omitempty"`
	WelcomeMessage       string `bson:"welcome_message,omitempty" json:"welcome_message,omitempty"`
}

// CustomDomain holds URL configuration for the employee-facing app
type CustomDomain struct {
	Slug         string `bson:"slug,omitempty" json:"slug,omitempty"`
	CustomDomain string `bson:"custom_domain,omitempty" json:"custom_domain,omitempty"`
	IsVerified   bool   `bson:"is_verified" json:"is_verified"`
	DNSStatus    string `bson:"dns_status,omitempty" json:"dns_status,omitempty"` // "pending", "active", "error"
	DNSError     string `bson:"dns_error,omitempty" json:"dns_error,omitempty"`
	SSLStatus    string `bson:"ssl_status,omitempty" json:"ssl_status,omitempty"` // "pending", "active", "error"
}

// CompanySettings holds configurable behavior for a company
type CompanySettings struct {
	// QuestionnaireVisibility controls what the company_admin sees:
	// "all" = full questionnaire catalog (default)
	// "assigned" = only questionnaires assigned to their company
	QuestionnaireVisibility string `bson:"questionnaire_visibility" json:"questionnaire_visibility"`
}

// Company represents a company entity
type Company struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name         string             `bson:"name" json:"name" validate:"required,min=3,max=200"`
	IsActive     bool               `bson:"is_active" json:"is_active"`
	Branding     *Branding          `bson:"branding,omitempty" json:"branding,omitempty"`
	CustomDomain *CustomDomain      `bson:"custom_domain,omitempty" json:"custom_domain,omitempty"`
	Settings     *CompanySettings   `bson:"settings,omitempty" json:"settings,omitempty"`
	CreatedAt    time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updated_at"`

	// FusionAuth multi-tenant fields
	FusionAuthTenantID      string `bson:"fusionauth_tenant_id,omitempty" json:"fusionauth_tenant_id,omitempty"`
	FusionAuthApplicationID string `bson:"fusionauth_application_id,omitempty" json:"fusionauth_application_id,omitempty"`
	FusionAuthClientID      string `bson:"fusionauth_client_id,omitempty" json:"fusionauth_client_id,omitempty"`
	FusionAuthClientSecret  string `bson:"fusionauth_client_secret,omitempty" json:"fusionauth_client_secret,omitempty"`

	// Subscription — plan, add-ons, and frozen price snapshot
	Subscription *CompanySubscription `bson:"subscription,omitempty" json:"subscription,omitempty"`

	// Transient field — not persisted, populated by service layer
	QuestionnaireCount int `bson:"-" json:"questionnaire_count"`
}

// CompanySubscription holds the contracted plan, add-ons and price snapshot
type CompanySubscription struct {
	PlanSlug      string        `bson:"plan_slug" json:"plan_slug"`
	PlanName      string        `bson:"plan_name" json:"plan_name"`
	AddOnSlugs    []string      `bson:"add_on_slugs" json:"add_on_slugs"`
	ContractType  string        `bson:"contract_type" json:"contract_type"` // "mensual" | "anual"
	PriceSnapshot PriceSnapshot `bson:"price_snapshot" json:"price_snapshot"`
	StartDate     *time.Time    `bson:"start_date,omitempty" json:"start_date,omitempty"`
	Notes         string        `bson:"notes,omitempty" json:"notes,omitempty"`
}

// PriceSnapshot freezes the pricing at the moment of assignment for billing
type PriceSnapshot struct {
	UfValueCLP      float64         `bson:"uf_value_clp" json:"uf_value_clp"`
	PlanPriceUf     float64         `bson:"plan_price_uf" json:"plan_price_uf"`
	AddOnsPriceUf   float64         `bson:"add_ons_price_uf" json:"add_ons_price_uf"`
	SetupFeeUf      float64         `bson:"setup_fee_uf" json:"setup_fee_uf"`
	DiscountPercent float64         `bson:"discount_percent" json:"discount_percent"`
	TotalMonthlyUf  float64         `bson:"total_monthly_uf" json:"total_monthly_uf"`
	TotalMonthlyCLP float64         `bson:"total_monthly_clp" json:"total_monthly_clp"`
	SnapshotDate    time.Time       `bson:"snapshot_date" json:"snapshot_date"`
	AddOnDetails    []AddOnSnapshot `bson:"add_on_details" json:"add_on_details"`
}

// AddOnSnapshot captures individual add-on pricing at snapshot time
type AddOnSnapshot struct {
	Slug    string  `bson:"slug" json:"slug"`
	Name    string  `bson:"name" json:"name"`
	PriceUf float64 `bson:"price_uf" json:"price_uf"`
}

// NewCompany creates a new Company with timestamps
func NewCompany(name string) *Company {
	now := time.Now()
	return &Company{
		ID:        primitive.NewObjectID(),
		Name:      name,
		IsActive:  true, // Default: true
		CreatedAt: now,
		UpdatedAt: now,
	}
}
