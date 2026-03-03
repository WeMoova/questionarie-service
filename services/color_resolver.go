package services

import (
	"context"

	"questionarie-service/models"
	"questionarie-service/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ResolvedColors holds the final resolved colors for report generation.
// Priority: CompanyQuestionnaire.ColorConfig > base EvaluationConfig.
type ResolvedColors struct {
	DimensionThresholdColors map[string]map[string]string // dim_code -> level -> hex
	RiskProfileColors        map[string]string            // profile_name -> hex
}

// resolveQuestionnaireColors resolves the final colors for reports.
// If the CompanyQuestionnaire has a ColorConfig with mode != "default", use it.
// Otherwise, fall back to colors from the base EvaluationConfig.
func (s *ReportService) resolveQuestionnaireColors(
	ctx context.Context,
	cq *models.CompanyQuestionnaire,
	evalConfig *models.EvaluationConfig,
) *ResolvedColors {
	// Check if the CQ has a non-default color config
	if cq.ColorConfig != nil && cq.ColorConfig.Mode != models.ColorModeDefault {
		return resolveFromColorConfig(cq.ColorConfig, evalConfig)
	}

	// Fallback: extract colors from the base questionnaire's EvaluationConfig
	return resolveFromEvalConfig(evalConfig)
}

// resolveFromColorConfig builds resolved colors from the CQ's ColorConfig.
func resolveFromColorConfig(cc *models.ColorConfig, evalConfig *models.EvaluationConfig) *ResolvedColors {
	resolved := &ResolvedColors{
		DimensionThresholdColors: make(map[string]map[string]string),
		RiskProfileColors:        make(map[string]string),
	}

	// Build lookup from dimension colors
	for _, dc := range cc.DimensionColors {
		resolved.DimensionThresholdColors[dc.DimensionCode] = dc.ThresholdColors
	}

	// Build lookup from risk profile colors
	for _, rpc := range cc.RiskProfileColors {
		resolved.RiskProfileColors[rpc.ProfileName] = rpc.Color
	}

	// Fill in any dimensions/profiles not covered by the ColorConfig (fallback to base)
	if evalConfig != nil {
		for _, dim := range evalConfig.Dimensions {
			if _, ok := resolved.DimensionThresholdColors[dim.Code]; !ok {
				threshColors := make(map[string]string)
				for _, t := range dim.Thresholds {
					if t.Color != "" {
						threshColors[t.Level] = t.Color
					}
				}
				if len(threshColors) > 0 {
					resolved.DimensionThresholdColors[dim.Code] = threshColors
				}
			}
		}
		for _, rp := range evalConfig.RiskProfiles {
			if _, ok := resolved.RiskProfileColors[rp.Name]; !ok {
				if rp.Color != "" {
					resolved.RiskProfileColors[rp.Name] = rp.Color
				}
			}
		}
	}

	return resolved
}

// resolveFromEvalConfig builds resolved colors from the base questionnaire config.
func resolveFromEvalConfig(evalConfig *models.EvaluationConfig) *ResolvedColors {
	resolved := &ResolvedColors{
		DimensionThresholdColors: make(map[string]map[string]string),
		RiskProfileColors:        make(map[string]string),
	}

	if evalConfig == nil {
		return resolved
	}

	for _, dim := range evalConfig.Dimensions {
		threshColors := make(map[string]string)
		for _, t := range dim.Thresholds {
			if t.Color != "" {
				threshColors[t.Level] = t.Color
			}
		}
		if len(threshColors) > 0 {
			resolved.DimensionThresholdColors[dim.Code] = threshColors
		}
	}

	for _, rp := range evalConfig.RiskProfiles {
		if rp.Color != "" {
			resolved.RiskProfileColors[rp.Name] = rp.Color
		}
	}

	return resolved
}

// getCompanyPrimaryColorForResolver is a helper to get a company's primary color.
func (s *ReportService) getCompanyPrimaryColorForResolver(ctx context.Context, companyID primitive.ObjectID) string {
	company, err := s.companyRepo.GetByID(ctx, companyID)
	if err == nil && company.Branding != nil && company.Branding.PrimaryColor != "" {
		return company.Branding.PrimaryColor
	}
	return utils.DefaultPrimaryColor
}
