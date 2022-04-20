E2E tests
=========

This directory holds the e2e tests for celvet. All files in this directory
matching the pattern `(*)_test.yaml` are considered to be test input files, with
a corresponding file  `(*)_output.txt` .

Adding a new test
-----------------

First write the YAML file that will be linted (and save as something like `crontab_test.yaml`):

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: crontabs.stable.example.com
spec:
  group: stable.example.com
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                cronSpec:
                  type: string
                image:
                  type: string
                replicas:
                  type: integer
  scope: Namespaced
  names:
    plural: crontabs
    singular: crontab
    kind: CronTab
    shortNames:
    - ct
```

Next, the output file (`crontab_output.txt`):

```
string "<root>.spec.image" missing maxLength
string "<root>.spec.cronSpec" missing maxLength
```

Note that the output file should include a trailing newline, as `celvet` will
print one (and failing to include a newline will cause a false test failure).

Running the tests
------------------

```sh
go test --tags=e2e
```