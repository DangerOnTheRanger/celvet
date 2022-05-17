/*
Copyright 2022 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package celvet

import (
	"fmt"

	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// SchemaType represents a list, map or string as used by LimitError.
type SchemaType int

const (
	// SchemaTypeList represents a list as used by LimitError.
	SchemaTypeList SchemaType = iota
	// SchemaTypeMap represents a map as used by LimitError.
	SchemaTypeMap
	// SchemaTypeString represents a string as used by LimitError.
	SchemaTypeString
)

// LimitError represents a list, map, or string that lacks a user-set limit.
// For lists, this means maxItems has not been set. For maps, this means
// maxProperties has not been set. And for strings this means maxLength has not
// been set.
type LimitError struct {
	// Path represents the path to the list, map or string without the limit.
	Path *field.Path
	// Type indicates the type of the schema node that caused the error.
	Type SchemaType
}

func (l *LimitError) Error() string {
	switch l.Type {
	case SchemaTypeList:
		return fmt.Sprintf("list %q missing maxItems", l.Path.String())
	case SchemaTypeMap:
		return fmt.Sprintf("map %q missing maxProperties", l.Path.String())
	case SchemaTypeString:
		return fmt.Sprintf("string %q missing maxLength", l.Path.String())
	}
	return ""
}

// CheckMaxLimits takes a schema and returns a list of linter errors
// for every missing limit that could be set on a list/map/string belonging
// to that schema or any level beneath it.
func CheckMaxLimits(schema *structuralschema.Structural) []*LimitError {
	return checkMaxLimits(schema, field.NewPath("openAPIV3Schema"))
}

func checkMaxLimits(schema *structuralschema.Structural, path *field.Path) []*LimitError {
	limitErrors := make([]*LimitError, 0)
	switch schema.Type {
	case "array":
		if schema.ValueValidation == nil {
			limitErrors = append(limitErrors, &LimitError{path, SchemaTypeList})
		} else if schema.ValueValidation.MaxItems == nil {
			limitErrors = append(limitErrors, &LimitError{path, SchemaTypeList})
		}
		limitErrors = append(limitErrors, checkMaxLimits(schema.Items, path.Child("<items>"))...)
	case "string":
		if schema.ValueValidation == nil {
			limitErrors = append(limitErrors, &LimitError{path, SchemaTypeString})
		} else if schema.ValueValidation.MaxLength == nil {
			limitErrors = append(limitErrors, &LimitError{path, SchemaTypeString})
		}
	case "object":
		if schema.AdditionalProperties != nil && schema.AdditionalProperties.Structural != nil {
			if schema.ValueValidation == nil {
				limitErrors = append(limitErrors, &LimitError{path, SchemaTypeMap})
			} else if schema.ValueValidation.MaxProperties == nil {
				limitErrors = append(limitErrors, &LimitError{path, SchemaTypeMap})
			}
			limitErrors = append(limitErrors, checkMaxLimits(schema.AdditionalProperties.Structural, path)...)
		}
		for propName, propSchema := range schema.Properties {
			limitErrors = append(limitErrors, checkMaxLimits(&propSchema, path.Child(propName))...)
		}
	}
	return limitErrors
}
