# TEMPORARY README FOR COMMANDS UNTIL CLI IS REINSTATED

# Installation

Install it with the `go install` command:

```sh
go install github.com/speakeasy-api/openapi-overlay@latest
```

# Usage

The tool provides sub-commands such as `apply`, `validate` and `compare` under the `openapi-overlay` command for working with overlay files.

The recommended usage pattern is through Speakeasy CLI command `speakeasy overlay`. Please see [here](https://www.speakeasyapi.dev/docs/speakeasy-cli/overlay/README) for CLI installation and usage documentation.

However, the `openapi-overlay` tool can be used standalone.

For more examples of usage, see [here](https://www.speakeasyapi.dev/docs/openapi/overlays)

## Apply

The most obvious use-case for this command is applying an overlay to a specification file.

```sh
openapi-overlay apply --overlay=overlay.yaml --schema=spec.yaml
```

If the overlay file has the `extends` key set to a `file://` URL, then the `spec.yaml` file may be omitted.

## Validate

A command is provided to perform basic validation of the overlay file itself. It will not tell you whether it will apply correctly or whether the application will generate a valid OpenAPI specification. Rather, it is limited to just telling you when the spec follows the OpenAPI Overlay Specification correctly: all required fields are present and have valid values.

```sh
openapi-overlay validate --overlay=overlay.yaml
```

## Compare

Finally, a tool is provided that will generate an OpenAPI Overlay specification from two input files.

```sh
openapi-overlay compare --before=spec1.yaml --after=spec2.yaml --out=overlay.yaml
```

the overlay file will be written to a file called `overlay.yaml` with a diagnostic output in the console.

# Other Notes

This tool works with either YAML or JSON input files, but always outputs YAML at this time.