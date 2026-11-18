module github.com/speakeasy-api/openapi/openapi/linter/customrules

go 1.26.7

replace github.com/speakeasy-api/openapi => ../../../

require (
	github.com/dop251/goja v0.0.0-20260106131823-651366fbe6e3
	github.com/evanw/esbuild v0.28.2
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible
	github.com/speakeasy-api/openapi v1.28.1-0.20261118220257-c9d1c24238dc
	github.com/stretchr/testify v1.12.1
	go.yaml.in/yaml/v3 v3.0.6
)

require (
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/google/pprof v0.0.0-20250403155104-27863c87afa6 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	golang.org/x/sync v0.25.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.44.0 // indirect
)
