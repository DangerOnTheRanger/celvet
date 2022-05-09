//+build e2e

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

package celvet_test

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/DangerOnTheRanger/celvet"
)

func TestE2E(t *testing.T) {
	testFilePaths, err := filepath.Glob("testdata/e2e/*_test.yaml")
	if err != nil {
		t.Fatal(err)
	}
	outputRegex := regexp.MustCompile(`testdata\/e2e\/(.*)_test.yaml`)
	for _, testFilePath := range testFilePaths {
		t.Run(testFilePath, func(t *testing.T) {
			outputFilePath := outputRegex.ReplaceAllString(testFilePath, `testdata/e2e/${1}_output.txt`)
			expectedOutputFileBytes, err := ioutil.ReadFile(outputFilePath)
			if err != nil {
				t.Fatal(err)
			}
			observedOutputBytes := new(bytes.Buffer)
			linterArgs := []string{
				"celvet",
				testFilePath,
			}
			_, err = celvet.Lint(linterArgs, observedOutputBytes)
			if err != nil {
				t.Fatal(err)
			}
			expectedOutput := string(expectedOutputFileBytes)
			t.Logf("expected output:\n%s", expectedOutput)
			observedOutput := observedOutputBytes.String()
			t.Logf("observed output:\n%s", observedOutput)
			if expectedOutput != observedOutput {
				t.Errorf("output mismatch: expected:\n%sgot:\n%s", expectedOutput, observedOutput)
			}
		})
	}
}
