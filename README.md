[![https://www.speakeasy.com](.github/assets/speakeasy.png?raw=true)](https://www.speakeasy.com)

# [github.com/speakeasy-api/openapi](https://github.com/speakeasy-api/openapi)

[![Reference](https://godoc.org/github.com/speakeasy-api/openapi?status.svg)](http://godoc.org/github.com/speakeasy-api/openapi)
![Pipeline](https://github.com/speakeasy-api/openapi/workflows/test/badge.svg)
[![GoReportCard](https://goreportcard.com/badge/github.com/speakeasy-api/openapi)](https://goreportcard.com/report/github.com/speakeasy-api/openapi)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

The Speakeasy OpenAPI module provides a set of packages and tools for working with OpenAPI Specification documents.

Used directly in Speakeasy's products it powers our [SDK Generator](https://www.speakeasy.com/docs/create-client-sdks) and [Contract Testing](https://www.speakeasy.com/docs/testing) tools.

Documentation for the packages can be found in the [GoDoc documentation.](https://pkg.go.dev/github.com/speakeasy-api/openapi)

## Main Packages

### [arazzo](./arazzo)

The `arazzo` package provides an API for working with Arazzo documents including reading, creating, mutating, walking and validating them.

## Sub Packages

This repository also contains a number of sub packages that are used by the main packages to provide the required functionality. The below packages may be moved into their own repository in the future, depending on future needs.

### [json](./json)

The `json` package provides utilities for converting between JSON and YAML.

### [jsonpointer](./jsonpointer)

The `jsonpointer` package provides an API for working with [RFC 6901](https://datatracker.ietf.org/doc/html/rfc6901) compliant JSON Pointers. Providing functionality for validating JSON Pointers, and extracting the target of a JSON Pointer for various Go types and structures.

### [jsonschema](./jsonschema)

The `jsonschema` package provides various models for working with the different JSON Schema dialects.

### [sequencedmap](./sequencedmap)

The `sequencedmap` package provides a map implementation that maintains the order of keys as they are added.

## Contributing

This repository is maintained by Speakeasy, but we welcome and encourage contributions from the community to help improve its capabilities and stability.

### How to Contribute

1. **Open Issues**: Found a bug or have a feature suggestion? Open an issue to describe what you'd like to see changed.

2. **Pull Requests**: We welcome pull requests! If you'd like to contribute code:
   - Fork the repository
   - Create a new branch for your feature/fix
   - Submit a PR with a clear description of the changes and any related issues

3. **Feedback**: Share your experience using the packages or suggest improvements.

All contributions, whether they're bug reports, feature requests, or code changes, help make this project better for everyone.

Please ensure your contributions adhere to our coding standards and include appropriate tests where applicable.
