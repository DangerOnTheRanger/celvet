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

func genStringSchema(maxLength *int64) *structuralschema.Structural {
	return &structuralschema.Structural{
		Generic: structuralschema.Generic{
			Type: "string",
		},
		ValueValidation: &structuralschema.ValueValidation{
			MaxLength: maxLength,
		},
	}
}

func withRule(schema *structuralschema.Structural, rule string) *structuralschema.Structural {
	schema.Extensions.XValidations = append(schema.Extensions.XValidations, apiextensions.ValidationRule{
		Rule: rule,
	})
	return schema
}

func genArraySchema(maxItems *int64, items *structuralschema.Structural) *structuralschema.Structural {
	return &structuralschema.Structural{
		Generic: structuralschema.Generic{
			Type: "array",
		},
		Items: items,
		ValueValidation: &structuralschema.ValueValidation{
			MaxItems: maxItems,
		},
	}
}

func genRootSchema(childName string, childSchema *structuralschema.Structural) *structuralschema.Structural {
	return &structuralschema.Structural{
		Generic: structuralschema.Generic{
			Type: "object",
		},
		Properties: map[string]structuralschema.Structural{
			childName: *childSchema,
		},
	}
}

func genMapSchema(maxProperties *int64, properties *structuralschema.Structural) *structuralschema.Structural {
	return &structuralschema.Structural{
		Generic: structuralschema.Generic{
			Type: "object",
			AdditionalProperties: &structuralschema.StructuralOrBool{
				Structural: properties,
			},
		},
		ValueValidation: &structuralschema.ValueValidation{
			MaxProperties: maxProperties,
		},
	}
}

func TestCost(t *testing.T) {
	tests := []struct {
		name                     string
		schema                   *structuralschema.Structural
		expectedErrors           []*CostError
		numExpectedCompileErrors int
	}{
		{
			name:   "array",
			schema: genRootSchema("array", withRule(genArraySchema(nil, genStringSchema(nil)), `self.all(x, x == x)`)),
			expectedErrors: []*CostError{
				{
					Path: "<root>.array",
					Cost: 329858626352,
				},
			},
		},
		{
			name:   "arrayWithItemExpression",
			schema: genRootSchema("array", genArraySchema(nil, withRule(genStringSchema(nil), `self == self`))),
			expectedErrors: []*CostError{
				{
					Path: "<root>.array.<items>",
					Cost: 329855795200,
				},
			},
		},
		{
			name:           "arrayWithSafeCost",
			schema:         genRootSchema("array", withRule(genArraySchema(int64ptr(5), genStringSchema(nil)), `self.all(x, x == x)`)),
			expectedErrors: []*CostError{},
		},
		{
			name:   "map",
			schema: withRule(genMapSchema(nil, genStringSchema(nil)), `self.all(x, self.all(y, x == y))`),
			expectedErrors: []*CostError{
				{
					Path: "<root>",
					Cost: 773092147202,
				},
			},
		},
		{
			name:           "mapWithSafeCost",
			schema:         withRule(genMapSchema(int64ptr(5), genStringSchema(nil)), `self.all(x, self.all(y, x == y))`),
			expectedErrors: []*CostError{},
		},
		{
			name:   "mapWithPropertyExpression",
			schema: genMapSchema(nil, withRule(genStringSchema(nil), `self == self`)),
			expectedErrors: []*CostError{
				{
					Path: "<root>.<properties>",
					Cost: 329855795200,
				},
			},
		},
		{
			name: "string",
			schema: genRootSchema("excessiveString", withRule(genStringSchema(nil),
				`["abc", "def", "ghi", "jhk"].all(x, ["abc", "def", "ghi", "jhk"].all(y, x == self && y == self && x == y))`)),
			expectedErrors: []*CostError{
				{
					Path: "<root>.excessiveString",
					Cost: 15099715,
				},
			},
		},
		{
			name:           "stringWithSafeCost",
			schema:         genRootSchema("safeString", withRule(genStringSchema(nil), `self == self`)),
			expectedErrors: []*CostError{},
		},
		{
			name:                     "compileError",
			schema:                   withRule(genStringSchema(nil), `self.all(x, true)`),
			numExpectedCompileErrors: 1,
		},
		{
			name:   "nestedSchemas",
			schema: genRootSchema("mapWithArray", genMapSchema(nil, genArraySchema(nil, withRule(genStringSchema(nil), `self == self`)))),
			expectedErrors: []*CostError{
				{
					Path: "<root>.mapWithArray.<properties>.<items>",
					Cost: 329855795200,
				},
			},
		},
		{
			name:   "multipleRules",
			schema: genRootSchema("multiRuleArray", withRule(genArraySchema(nil, withRule(genStringSchema(nil), `true`)), `self.all(x, self.all(y, x == y))`)),
			expectedErrors: []*CostError{
				{
					Path: "<root>.multiRuleArray",
					Cost: 345881509130194127,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			costErrors, compileErrors := CheckExprCost(test.schema)
			if len(compileErrors) != test.numExpectedCompileErrors {
				t.Errorf("Unexpected number of compile errors (got %d, expected %d)", len(compileErrors), test.numExpectedCompileErrors)
			}
			if len(costErrors) != len(test.expectedErrors) {
				t.Errorf("Wrong number of expected errors (got %v, expected %v)", costErrors, test.expectedErrors)
			}
			for i, seenError := range costErrors {
				expectedError := test.expectedErrors[i]
				if !errorsEqual(seenError, expectedError) {
					t.Errorf("Wrong error (expected %v, got %v)", expectedError, seenError)
				}
			}
		})
	}
}

func errorsEqual(x, y *CostError) bool {
	return x.Path == y.Path && x.Cost == y.Cost
}

func int64ptr(i int64) *int64 {
	return &i
}
