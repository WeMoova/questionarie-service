package models

import "time"

// DimensionScore holds the calculated score for a single dimension
type DimensionScore struct {
	DimensionCode string  `bson:"dimension_code" json:"dimension_code"`
	DimensionName string  `bson:"dimension_name" json:"dimension_name"`
	RawScore      float64 `bson:"raw_score" json:"raw_score"`
	MaxScore      int     `bson:"max_score" json:"max_score"`
	Level         string  `bson:"level" json:"level"`             // "alto", "medio", "bajo"
	LevelLabel    string  `bson:"level_label" json:"level_label"` // "ALTO", "MEDIO", "BAJO"
	LevelColor    string  `bson:"level_color" json:"level_color"`
	Direction     string  `bson:"direction" json:"direction"` // "direct", "inverse"
}

// EvaluationResult stores the calculated evaluation for a completed assignment
type EvaluationResult struct {
	DimensionScores       []DimensionScore `bson:"dimension_scores" json:"dimension_scores"`
	GeneralInterpretation string           `bson:"general_interpretation,omitempty" json:"general_interpretation,omitempty"`
	EvaluatedAt           time.Time        `bson:"evaluated_at" json:"evaluated_at"`
}
