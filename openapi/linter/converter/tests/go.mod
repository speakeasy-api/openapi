module github.com/speakeasy-api/openapi/openapi/linter/converter/tests

go 1.26.7

replace (
	github.com/speakeasy-api/openapi => ../../../..
	github.com/speakeasy-api/openapi/openapi/linter/customrules => ../../customrules
)

require (
	github.com/speakeasy-api/openapi v1.26.4-0.20261116012745-bdb0d271330e
	github.com/speakeasy-api/openapi/openapi/linter/customrules v0.0.0-20261116012745-bdb0d271330e
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/dop251/goja v0.0.0-20260106131823-651366fbe6e3 // indirect
	github.com/evanw/esbuild v0.28.2 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/google/pprof v0.0.0-20250403155104-27863c87afa6 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	go.yaml.in/yaml/v3 v3.0.6 // indirect
	golang.org/x/sync v0.25.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.44.0 // indirect
)
