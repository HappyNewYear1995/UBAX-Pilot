package datacore

import (
	"github.com/ubax/ubax-pilot/pkg/logger"
)

// Transform defines a data transformation rule
type Transform struct {
	Type   string                 `json:"type"`   // "mask", "extract", "convert", etc.
	Field  string                 `json:"field"`  // Target field name
	Config map[string]interface{} `json:"config"` // Transform-specific config
}

// DataTransformer applies edge computing and data cleaning before sending to server
type DataTransformer struct {
	transforms []Transform
}

// NewDataTransformer creates a new data transformer
func NewDataTransformer(transforms []Transform) *DataTransformer {
	return &DataTransformer{
		transforms: transforms,
	}
}

// Apply applies all transforms to the data
func (dt *DataTransformer) Apply(data map[string]interface{}) map[string]interface{} {
	result := data
	for _, t := range dt.transforms {
		result = dt.applyTransform(result, t)
	}
	return result
}

func (dt *DataTransformer) applyTransform(data map[string]interface{}, t Transform) map[string]interface{} {
	switch t.Type {
	case "mask":
		return dt.maskField(data, t.Field)
	case "extract":
		return dt.extractField(data, t)
	case "convert":
		return dt.convertField(data, t)
	default:
		logger.Warn("Unknown transform type:", t.Type)
		return data
	}
}

// maskField masks sensitive data (e.g., PII, passwords)
func (dt *DataTransformer) maskField(data map[string]interface{}, field string) map[string]interface{} {
	if val, ok := data[field]; ok {
		if str, ok := val.(string); ok {
			if len(str) > 4 {
				data[field] = "****" + str[len(str)-4:]
			} else {
				data[field] = "****"
			}
		}
	}
	return data
}

// extractField extracts specific fields from the data
func (dt *DataTransformer) extractField(data map[string]interface{}, t Transform) map[string]interface{} {
	// TODO: Implement field extraction based on regex or JSONPath
	return data
}

// convertField converts field format (e.g., timestamp, log level normalization)
func (dt *DataTransformer) convertField(data map[string]interface{}, t Transform) map[string]interface{} {
	// TODO: Implement field conversion
	return data
}
