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
	"math"

	"github.com/google/cel-go/cel"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/validation"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	schemacel "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel"
)

type costError struct {
	Program cel.Program
	Name    string
	Cost    uint64
}

func (c *costError) Error() string {
	exceedFactor := float64(c.Cost) / float64(validation.StaticEstimatedCostLimit)
	return fmt.Sprintf("expression at %q exceeded budget by factor of %.1fx", c.Name, exceedFactor)
}

func CheckExprCost(schema *structuralschema.Structural) []*costError {
	// TODO(DangerOnTheRanger): swap out name system for fieldpaths
	return checkExprCost(schema, "<root>", rootCostInfo())
}

func checkExprCost(schema *structuralschema.Structural, name string, nodeCostInfo costInfo) []*costError {
	results, err := schemacel.Compile(schema, false, schemacel.PerCallLimit)
	if err != nil {
		// TODO(DangerOnTheRanger): what's the proper way to handle compilation errors here?
		fmt.Printf("error during compilation at %q: %s\n", name, err)
		return nil
	}
	costErrors := make([]*costError, 0)
	for _, result := range results {
		exprCost := getExpressionCost(result, nodeCostInfo)
		if exprCost > validation.StaticEstimatedCostLimit {
			costErrors = append(costErrors, &costError{Program: result.Program,
				Name: name,
				Cost: exprCost,
			})
		}
	}

	switch schema.Type {
	case "array":
		costErrors = append(costErrors, checkExprCost(schema.Items, name+".<items>", nodeCostInfo.MultiplyByElementCost(schema))...)
	case "object":
		for propName, propSchema := range schema.Properties {
			costErrors = append(costErrors, checkExprCost(&propSchema, name+"."+propName, nodeCostInfo.MultiplyByElementCost(schema))...)
		}
	}
	return costErrors
}

// code below is copied from k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/validation/validation.go
// ideally some/all of these symbols should be exported from there instead,
// though slight modifications have been made to use structural schemas (and
// not JSONSchemaProps)

func getExpressionCost(cr schemacel.CompilationResult, cardinalityCost costInfo) uint64 {
	if cardinalityCost.MaxCardinality != unbounded {
		return multiplyWithOverflowGuard(cr.MaxCost, *cardinalityCost.MaxCardinality)
	}
	return multiplyWithOverflowGuard(cr.MaxCost, cr.MaxCardinality)
}

// multiplyWithOverflowGuard returns the product of baseCost and cardinality unless that product
// would exceed math.MaxUint, in which case math.MaxUint is returned.
func multiplyWithOverflowGuard(baseCost, cardinality uint64) uint64 {
	if baseCost == 0 {
		// an empty rule can return 0, so guard for that here
		return 0
	} else if math.MaxUint/baseCost < cardinality {
		return math.MaxUint
	}
	return baseCost * cardinality
}

// unbounded uses nil to represent an unbounded cardinality value.
var unbounded *uint64 = nil

type costInfo struct {
	// MaxCardinality represents a limit to the number of data elements that can exist for the current
	// schema based on MaxProperties or MaxItems limits present on parent schemas, If all parent
	// map and array schemas have MaxProperties or MaxItems limits declared MaxCardinality is
	// an int pointer representing the product of these limits.  If least one parent map or list schema
	// does not have a MaxProperties or MaxItems limits set, the MaxCardinality is nil, indicating
	// that the parent schemas offer no bound to the number of times a data element for the current
	// schema can exist.
	MaxCardinality *uint64
}

// MultiplyByElementCost returns a costInfo where the MaxCardinality is multiplied by the
// factor that the schema increases the cardinality of its children. If the costInfo's
// MaxCardinality is unbounded (nil) or the factor that the schema increase the cardinality
// is unbounded, the resulting costInfo's MaxCardinality is also unbounded.
func (c *costInfo) MultiplyByElementCost(schema *structuralschema.Structural) costInfo {
	result := costInfo{MaxCardinality: unbounded}
	if schema == nil {
		// nil schemas can be passed since we call MultiplyByElementCost
		// before ValidateCustomResourceDefinitionOpenAPISchema performs its nil check
		return result
	}
	if c.MaxCardinality == unbounded {
		return result
	}
	maxElements := extractMaxElements(schema)
	if maxElements == unbounded {
		return result
	}
	result.MaxCardinality = uint64ptr(multiplyWithOverflowGuard(*c.MaxCardinality, *maxElements))
	return result
}

// extractMaxElements returns the factor by which the schema increases the cardinality
// (number of possible data elements) of its children.  If schema is a map and has
// MaxProperties or an array has MaxItems, the int pointer of the max value is returned.
// If schema is a map or array and does not have MaxProperties or MaxItems,
// unbounded (nil) is returned to indicate that there is no limit to the possible
// number of data elements imposed by the current schema.  If the schema is an object, 1 is
// returned to indicate that there is no increase to the number of possible data elements
// for its children.  Primitives do not have children, but 1 is returned for simplicity.
func extractMaxElements(schema *structuralschema.Structural) *uint64 {
	switch schema.Type {
	case "object":
		if schema.AdditionalProperties != nil {
			if schema.ValueValidation != nil && schema.ValueValidation.MaxProperties != nil {
				maxProps := uint64(zeroIfNegative(*schema.ValueValidation.MaxProperties))
				return &maxProps
			}
			return unbounded
		}
		// return 1 to indicate that all fields of an object exist at most one for
		// each occurrence of the object they are fields of
		return uint64ptr(1)
	case "array":
		if schema.ValueValidation != nil && schema.ValueValidation.MaxItems != nil {
			maxItems := uint64(zeroIfNegative(*schema.ValueValidation.MaxItems))
			return &maxItems
		}
		return unbounded
	default:
		return uint64ptr(1)
	}
}

func zeroIfNegative(v int64) int64 {
	if v < 0 {
		return 0
	}
	return v
}

func uint64ptr(i uint64) *uint64 {
	return &i
}

func rootCostInfo() costInfo {
	rootCardinality := uint64(1)
	return costInfo{
		MaxCardinality: &rootCardinality,
	}
}
