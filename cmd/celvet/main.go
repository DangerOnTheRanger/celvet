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

package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/DangerOnTheRanger/celvet"
	api "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiinstall "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	apiv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apimachinery/pkg/runtime"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s crd-file\n", os.Args[0])
		os.Exit(1)
	}

	crdFile := os.Args[1]
	fileBytes, err := ioutil.ReadFile(crdFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading %s: %s\n", crdFile, err)
		os.Exit(1)
	}
	scheme := runtime.NewScheme()
	apiinstall.Install(scheme)
	codecs := runtimeserializer.NewCodecFactory(scheme)
	decode := codecs.UniversalDeserializer().Decode
	obj, _, err := decode(fileBytes, nil, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error while decoding: %s\n", err)
		os.Exit(1)
	}
	switch obj.(type) {
	case *apiv1.CustomResourceDefinition:
	default:
		fmt.Fprintf(os.Stderr, "unexpected decoded object (expected CustomResourceDefinition), got %T\n", obj)
		os.Exit(1)
	}
	spec := obj.(*apiv1.CustomResourceDefinition).Spec
	// TODO(DangerOnTheRanger): support multiple CRD versions
	v1Schema := spec.Versions[0].Schema.OpenAPIV3Schema
	schema := &api.JSONSchemaProps{}
	err = apiv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(v1Schema, schema, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error during schema conversion: %s\n", err)
		os.Exit(1)
	}
	structural, err := structuralschema.NewStructural(schema)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error converting to structural schema: %s\n", err)
		os.Exit(1)
	}

	lintExitStatus := 0
	limitErrors := celvet.CheckMaxLimits(structural)
	if len(limitErrors) != 0 {
		for _, lintError := range limitErrors {
			os.Stderr.WriteString(lintError.Error() + "\n")
		}
		lintExitStatus = 1
	}

	costErrors, compileErrors := celvet.CheckExprCost(structural)
	if len(costErrors) != 0 {
		for _, lintError := range costErrors {
			fmt.Fprintf(os.Stderr, "%s\n", lintError.Error())
		}
		lintExitStatus = 1
	}
	if len(compileErrors) != 0 {
		for _, compileError := range compileErrors {
			os.Stderr.WriteString(compileError.Error() + "\n")
		}
		lintExitStatus = 1
	}

	os.Exit(lintExitStatus)
}
