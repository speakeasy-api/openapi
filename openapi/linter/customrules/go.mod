module github.com/speakeasy-api/openapi/openapi/linter/customrules

go 1.26.0

replace github.com/speakeasy-api/openapi => ../../../

require (
	github.com/dop251/goja v0.0.0-20260106131823-651366fbe6e3
	github.com/evanw/esbuild v0.28.2
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible
	github.com/speakeasy-api/openapi v1.25.7-0.20261016003642-daec030de9fa
	github.com/stretchr/testify v1.12.1
	go.yaml.in/yaml/v3 v3.0.5
)

require (
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.43.0 // indirect
)
