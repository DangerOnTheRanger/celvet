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
		fmt.Printf("usage: %s crd-file\n", os.Args[0])
		os.Exit(1)
	}

	crdFile := os.Args[1]
	fileBytes, err := ioutil.ReadFile(crdFile)
	if err != nil {
		fmt.Printf("error reading %s: %s\n", crdFile, err)
		os.Exit(1)
	}
	scheme := runtime.NewScheme()
	apiinstall.Install(scheme)
	codecs := runtimeserializer.NewCodecFactory(scheme)
	decode := codecs.UniversalDeserializer().Decode
	obj, _, err := decode(fileBytes, nil, nil)
	if err != nil {
		fmt.Printf("error while decoding: %s\n", err)
		os.Exit(1)
	}
	switch obj.(type) {
	case *apiv1.CustomResourceDefinition:
	default:
		fmt.Printf("unexpected decoded object (expected CustomResourceDefinition), got %T\n", obj)
		os.Exit(1)
	}
	spec := obj.(*apiv1.CustomResourceDefinition).Spec
	// TODO(DangerOnTheRanger): support multiple CRD versions
	v1Schema := spec.Versions[0].Schema.OpenAPIV3Schema
	schema := &api.JSONSchemaProps{}
	err = apiv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(v1Schema, schema, nil)
	if err != nil {
		fmt.Printf("error during schema conversion: %s\n", err)
		os.Exit(1)
	}
	structural, err := structuralschema.NewStructural(schema)
	if err != nil {
		fmt.Printf("error converting to structural schema: %s\n", err)
		os.Exit(1)
	}

	limitErrors := celvet.CheckMaxLimits(structural)
	if len(limitErrors) != 0 {
		for _, lintError := range limitErrors {
			fmt.Printf("%s\n", lintError)
		}
		os.Exit(1)
	}
}
