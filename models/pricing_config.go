package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PricingPlan struct {
	Slug             string `bson:"slug" json:"slug"`
	Name             string `bson:"name" json:"name"`
	Emoji            string `bson:"emoji" json:"emoji"`
	MaxEmployees     *int   `bson:"max_employees,omitempty" json:"max_employees,omitempty"` // nil = unlimited
	Questionnaires   string `bson:"questionnaires" json:"questionnaires"`
	ResponsesPerMonth string `bson:"responses_per_month" json:"responses_per_month"`
	PriceUfMonthly   *float64 `bson:"price_uf_monthly,omitempty" json:"price_uf_monthly,omitempty"` // nil = "A cotizar"
	Highlighted      bool     `bson:"highlighted,omitempty" json:"highlighted,omitempty"`
	Features         []string `bson:"features" json:"features"`
}

type PricingAddOn struct {
	Slug           string  `bson:"slug" json:"slug"`
	Name           string  `bson:"name" json:"name"`
	Emoji          string  `bson:"emoji" json:"emoji"`
	PriceUfMonthly float64 `bson:"price_uf_monthly" json:"price_uf_monthly"`
	Description    string  `bson:"description" json:"description"`
}

type PricingSetupFee struct {
	Slug        string  `bson:"slug" json:"slug"`
	Name        string  `bson:"name" json:"name"`
	Description string  `bson:"description" json:"description"`
	PriceUf     float64 `bson:"price_uf" json:"price_uf"`
	PriceCLP    float64 `bson:"price_clp" json:"price_clp"`
	Hours       int     `bson:"hours" json:"hours"`
	Category    string  `bson:"category" json:"category"` // "plan-setup" | "custom-service"
}

type PricingFaqItem struct {
	Question string `bson:"question" json:"question"`
	Answer   string `bson:"answer" json:"answer"`
}

type PricingConfig struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UfValueCLP      float64            `bson:"uf_value_clp" json:"uf_value_clp"`
	AnnualDiscount  float64            `bson:"annual_discount" json:"annual_discount"`
	VolumeDiscounts []VolumeDiscount   `bson:"volume_discounts" json:"volume_discounts"`
	Plans           []PricingPlan      `bson:"plans" json:"plans"`
	AddOns          []PricingAddOn     `bson:"add_ons" json:"add_ons"`
	SetupFees       []PricingSetupFee  `bson:"setup_fees" json:"setup_fees"`
	FaqItems        []PricingFaqItem   `bson:"faq_items" json:"faq_items"`
	UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
	UpdatedBy       string             `bson:"updated_by" json:"updated_by"`
}

type VolumeDiscount struct {
	MinEmployees int     `bson:"min_employees" json:"min_employees"`
	Discount     float64 `bson:"discount" json:"discount"`
}
