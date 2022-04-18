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
)

type schemaType int

const (
	typeList schemaType = iota
	typeMap
	typeString
)

type limitError struct {
	Name string
	Type schemaType
}

func (l *limitError) Error() string {
	switch l.Type {
	case typeList:
		return fmt.Sprintf("list %q missing maxItems", l.Name)
	case typeMap:
		return fmt.Sprintf("map %q missing maxProperties", l.Name)
	case typeString:
		return fmt.Sprintf("string %q missing maxLength", l.Name)
	}
	return ""
}

// CheckMaxLimits takes a schema and returns a list of linter errors
// for every missing limit that could be set on a list/map/string belonging
// to that schema or any level beneath it.
func CheckMaxLimits(schema *structuralschema.Structural) []error {
	return checkMaxLimits(schema, "<root>")
}

func checkMaxLimits(schema *structuralschema.Structural, name string) []error {
	limitErrors := make([]error, 0)
	switch schema.Type {
	case "array":
		if schema.ValueValidation == nil {
			limitErrors = append(limitErrors, &limitError{name, typeList})
		} else if schema.ValueValidation.MaxItems == nil {
			limitErrors = append(limitErrors, &limitError{name, typeList})
		}
		limitErrors = append(limitErrors, checkMaxLimits(schema.Items, name+".<items>")...)
	case "string":
		if schema.ValueValidation == nil {
			limitErrors = append(limitErrors, &limitError{name, typeString})
		} else if schema.ValueValidation.MaxLength == nil {
			limitErrors = append(limitErrors, &limitError{name, typeString})
		}
	case "object":
		if schema.AdditionalProperties != nil && schema.AdditionalProperties.Structural != nil {
			if schema.ValueValidation == nil {
				limitErrors = append(limitErrors, &limitError{name, typeMap})
			} else if schema.ValueValidation.MaxProperties == nil {
				limitErrors = append(limitErrors, &limitError{name, typeMap})
			}
			limitErrors = append(limitErrors, checkMaxLimits(schema.AdditionalProperties.Structural, name)...)
		}
		for propName, propSchema := range schema.Properties {
			limitErrors = append(limitErrors, checkMaxLimits(&propSchema, name+"."+propName)...)
		}
	}
	return limitErrors
}
