package consumer

import (
	"encoding/json"
)

// AttributeChecker is an interface for checking if an attribute is empty.
type AttributeChecker interface {
	IsEmpty() bool
}

// JSONAttributeChecker is an implementation of Checker for JSON data.
type JSONAttributeChecker struct {
	data []byte
}

// IsEmpty returns whether the data of the JSON data is empty
func (j *JSONAttributeChecker) IsEmpty() bool {
	if j.data == nil {
		return true
	}

	var jsonData interface{}
	if err := json.Unmarshal(j.data, &jsonData); err != nil {
		return false
	}

	switch v := jsonData.(type) {
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	default:
		return false
	}
}
