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

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
)

func TestCost(t *testing.T) {
	tests := []struct {
		name           string
		schema         *structuralschema.Structural
		expectedErrors []*costError
	}{
		{
			name: "array",
			schema: &structuralschema.Structural{
				Generic: structuralschema.Generic{
					Type: "object",
				},
				Properties: map[string]structuralschema.Structural{
					"array": structuralschema.Structural{
						Generic: structuralschema.Generic{
							Type: "array",
						},
						Items: &structuralschema.Structural{
							Generic: structuralschema.Generic{
								Type: "string",
							},
						},
						Extensions: structuralschema.Extensions{
							XValidations: apiextensions.ValidationRules{
								{
									Rule: `self.all(x, x == x)`,
								},
							},
						},
					},
				},
			},
			expectedErrors: []*costError{
				{
					Name: "<root>.array",
					Cost: 329858626352,
				},
			},
		},
		{
			name: "arrayWithLimit",
			schema: &structuralschema.Structural{
				Generic: structuralschema.Generic{
					Type: "array",
				},
				Items: &structuralschema.Structural{
					Generic: structuralschema.Generic{
						Type: "string",
					},
				},
				Extensions: structuralschema.Extensions{
					XValidations: apiextensions.ValidationRules{
						{
							Rule: `self.all(x, x == x)`,
						},
					},
				},
				ValueValidation: &structuralschema.ValueValidation{
					MaxItems: int64ptr(5),
				},
			},
			expectedErrors: []*costError{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errors := CheckExprCost(test.schema)
			if len(errors) != len(test.expectedErrors) {
				t.Errorf("Wrong number of expected errors (got %v, expected %v)", errors, test.expectedErrors)
			}
			for i, seenError := range errors {
				expectedError := test.expectedErrors[i]
				if !errorsEqual(seenError, expectedError) {
					t.Errorf("Wrong error (expected %v, got %v)", expectedError, seenError)
				}
			}
		})
	}
}

func errorsEqual(x, y *costError) bool {
	return x.Name == y.Name && x.Cost == y.Cost
}

func int64ptr(i int64) *int64 {
	return &i
}
