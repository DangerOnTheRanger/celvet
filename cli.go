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
	"io"
	"io/ioutil"

	api "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiinstall "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
)

func Lint(args []string, outputWriter io.Writer) (int, error) {
	if len(args) != 2 {
		return 1, fmt.Errorf("usage: %s crd-file", args[0])
	}

	crdFile := args[1]
	fileBytes, err := ioutil.ReadFile(crdFile)
	if err != nil {
		return 1, fmt.Errorf("error reading %s: %w", crdFile, err)
	}

	scheme := runtime.NewScheme()
	apiinstall.Install(scheme)
	codecs := runtimeserializer.NewCodecFactory(scheme)
	decode := codecs.UniversalDeserializer().Decode
	obj, _, err := decode(fileBytes, nil, nil)
	if err != nil {
		return 1, fmt.Errorf("error while decoding: %w", err)
	}
	switch obj.(type) {
	case *apiv1.CustomResourceDefinition:
	default:
		return 1, fmt.Errorf("unexpected decoded object (expected CustomResourceDefinition), got %T", obj)
	}

	spec := obj.(*apiv1.CustomResourceDefinition).Spec
	// TODO(DangerOnTheRanger): support multiple CRD versions
	v1Schema := spec.Versions[0].Schema.OpenAPIV3Schema
	schema := &api.JSONSchemaProps{}
	err = apiv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(v1Schema, schema, nil)
	if err != nil {
		return 1, fmt.Errorf("error during schema conversion: %w", err)
	}
	structural, err := structuralschema.NewStructural(schema)
	if err != nil {
		return 1, fmt.Errorf("error converting to structural schema: %w", err)
	}

	lintExitStatus := 0
	limitErrors := CheckMaxLimits(structural)
	if len(limitErrors) != 0 {
		for _, lintError := range limitErrors {
			fmt.Fprintln(outputWriter, lintError)
		}
		lintExitStatus = 1
	}

	costErrors := CheckExprCost(structural)
	if len(costErrors) != 0 {
		for _, lintError := range costErrors {
			fmt.Fprintln(outputWriter, lintError.Error())
		}
		lintExitStatus = 1
	}

	return lintExitStatus, nil
}
