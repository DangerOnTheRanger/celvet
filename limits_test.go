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
	"testing"

	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestMaxLimits(t *testing.T) {
	tests := []struct {
		name           string
		schema         *structuralschema.Structural
		expectedErrors []*LimitError
	}{
		{
			name: "integer",
			schema: &structuralschema.Structural{
				Generic: structuralschema.Generic{
					Type: "integer",
				},
			},
			expectedErrors: make([]*LimitError, 0),
		},
		{
			name: "missing array limit",
			schema: &structuralschema.Structural{
				Generic: structuralschema.Generic{
					Type: "array",
				},
				Items: &structuralschema.Structural{
					Generic: structuralschema.Generic{
						Type: "number",
					},
				},
			},
			expectedErrors: []*LimitError{
				{
					Path: field.NewPath("openAPIV3Schema"),
					Type: SchemaTypeList,
				},
			},
		},
		{
			name: "missing map limit",
			schema: &structuralschema.Structural{
				Generic: structuralschema.Generic{
					Type: "object",
					AdditionalProperties: &structuralschema.StructuralOrBool{
						Structural: &structuralschema.Structural{
							Generic: structuralschema.Generic{
								Type: "integer",
							},
						},
					},
				},
			},
			expectedErrors: []*LimitError{
				{
					Path: field.NewPath("openAPIV3Schema"),
					Type: SchemaTypeMap,
				},
			},
		},
		{
			name: "missing string limit",
			schema: &structuralschema.Structural{
				Generic: structuralschema.Generic{
					Type: "string",
				},
			},
			expectedErrors: []*LimitError{
				{
					Path: field.NewPath("openAPIV3Schema"),
					Type: SchemaTypeString,
				},
			},
		},
		{
			name: "missing map and string limit",
			schema: &structuralschema.Structural{
				Generic: structuralschema.Generic{
					Type: "object",
					AdditionalProperties: &structuralschema.StructuralOrBool{
						Structural: &structuralschema.Structural{
							Generic: structuralschema.Generic{
								Type: "string",
							},
						},
					},
				},
			},
			expectedErrors: []*LimitError{
				{
					Path: field.NewPath("openAPIV3Schema"),
					Type: SchemaTypeMap,
				},
				{
					Path: field.NewPath("openAPIV3Schema"),
					Type: SchemaTypeString,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errors := CheckMaxLimits(test.schema)
			if len(errors) != len(test.expectedErrors) {
				t.Errorf("Wrong number of expected errors (got %v, expected %v)", errors, test.expectedErrors)
			}
			for i, seenError := range errors {
				expectedError := test.expectedErrors[i]
				if !limitErrorsEqual(seenError, expectedError) {
					t.Errorf("Wrong error (expected %v, got %v)", expectedError, seenError)
				}
			}
		})
	}
}

func limitErrorsEqual(x, y *LimitError) bool {
	return x.Path.String() == y.Path.String() && x.Type == y.Type
}
