module github.com/speakeasy-api/openapi/openapi/linter/converter/tests

go 1.24.3

replace (
	github.com/speakeasy-api/openapi => ../../../..
	github.com/speakeasy-api/openapi/openapi/linter/customrules => ../../customrules
)

require (
	github.com/speakeasy-api/openapi v1.19.1-0.20260217225223-7d484a30828f
	github.com/speakeasy-api/openapi/openapi/linter/customrules v0.0.0-20260217225223-7d484a30828f
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dlclark/regexp2 v1.11.4 // indirect
	github.com/dop251/goja v0.0.0-20260106131823-651366fbe6e3 // indirect
	github.com/evanw/esbuild v0.27.2 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/google/pprof v0.0.0-20230207041349-798e818bf904 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	go.yaml.in/yaml/v4 v4.0.0-rc.4 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.36.0 // indirect
	golang.org/x/text v0.34.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
