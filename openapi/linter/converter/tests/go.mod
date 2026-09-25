module github.com/speakeasy-api/openapi/openapi/linter/converter/tests

go 1.26.0

replace (
	github.com/speakeasy-api/openapi => ../../../..
	github.com/speakeasy-api/openapi/openapi/linter/customrules => ../../customrules
)

require (
	github.com/speakeasy-api/openapi v1.25.2-0.20260921020105-543f9d00a250
	github.com/speakeasy-api/openapi/openapi/linter/customrules v0.0.0-20260921020105-543f9d00a250
	github.com/stretchr/testify v1.12.1
)

require (
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/dop251/goja v0.0.0-20260106131823-651366fbe6e3 // indirect
	github.com/evanw/esbuild v0.28.2 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
