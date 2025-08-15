<div align="center">
 <a href="https://www.speakeasy.com/" target="_blank">
  <img width="1500" height="500" alt="Speakeasy" src="https://github.com/user-attachments/assets/0e56055b-02a3-4476-9130-4be299e5a39c" />
 </a>
 <br />
 <br />
  <div>
   <a href="https://speakeasy.com/docs/create-client-sdks/" target="_blank"><b>Docs Quickstart</b></a>&nbsp;&nbsp;//&nbsp;&nbsp;<a href="https://go.speakeasy.com/slack" target="_blank"><b>Join us on Slack</b></a>
  </div>
 <br />

</div>

<hr />

<p align="center">
  <p align="center">
    <img  width="200px" alt="OpenAPI" src="https://github.com/user-attachments/assets/555a0899-5719-42ee-b4b1-ece8d1d812ea">
  </p>
  <h1 align="center"><b>OpenAPI</b></h1>
  <p align="center">A set of packages and tools for working with <a href="https://www.speakeasy.com/openapi">OpenAPI Specification documents</a>. <br /> Used directly in Speakeasy's product to power our <a href="https://www.speakeasy.com/docs/create-client-sdks">SDK Generator</a> and <a href="https://www.speakeasy.com/docs/testing">Contract Testing</a> tools.

</p>
  <p align="center">
    <!-- Badges -->
    <!-- OpenAPI Hub Badge -->
    <a href="https://www.speakeasy.com/openapi"><img src="https://custom-icon-badges.demolab.com/badge/-OpenAPI%20Hub-212015?style=for-the-badge&logoColor=FBE331&logo=speakeasy&labelColor=545454" /></a>
    <!-- OpenAPI Support Badge -->
    <a href="https://www.speakeasy.com/openapi"><img alt="OpenAPI Support" src="https://img.shields.io/badge/OpenAPI-3.0%20%7C%203.1-85EA2D.svg?style=for-the-badge&logo=openapiinitiative"></a>
    <!-- Arazzo Support Badge -->
    <img alt="Arazzo Support" src="https://img.shields.io/badge/Arazzo-1.0-purple.svg?style=for-the-badge">
    <a href="https://pkg.go.dev/github.com/speakeasy-api/openapi?tab=doc"><img alt="Go Doc" src="https://img.shields.io/badge/godoc-reference-blue.svg?style=for-the-badge"></a>
    <!-- Line Break --><br/>
    <!-- Release Version Badge -->
    <a href="https://github.com/speakeasy-api/openapi/releases/latest"><img alt="Release" src="https://img.shields.io/github/release/speakeasy-api/openapi.svg?style=for-the-badge"></a>
    <!-- Go Report Card Badge -->
    <a href="https://goreportcard.com/report/github.com/speakeasy-api/openapi"><img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/speakeasy-api/openapi?style=for-the-badge"></a>
    <!-- Security Badge -->
    <a href="https://github.com/speakeasy-api/openapi/actions/workflows/ci.yaml"><img alt="Security" src="https://img.shields.io/badge/security-scanned-green.svg?style=for-the-badge&logo=security"></a>
    <!-- CI Badge -->
    <a href="https://github.com/speakeasy-api/openapi/actions/workflows/ci.yaml"><img alt="GitHub Action: CI" src="https://img.shields.io/github/actions/workflow/status/speakeasy-api/openapi/ci.yaml?style=for-the-badge"></a>
    <!-- Line Break --><br/>
    <!-- Go Version Badge -->
    <a href="https://golang.org/"><img alt="Go Version" src="https://img.shields.io/badge/go-1.24.3+-00ADD8.svg?style=for-the-badge&logo=go"></a>
    <!-- Platform Support Badge -->
    <img alt="Platform Support" src="https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey.svg?style=for-the-badge">
    <!-- Stars Badge -->
    <a href="https://github.com/speakeasy-api/openapi/stargazers"><img alt="GitHub stars" src="https://img.shields.io/github/stars/speakeasy-api/openapi.svg?style=for-the-badge&logo=github"></a>
    <!-- Line Break --><br/>
    <!-- Built By Speakeasy Badge -->
    <a href="https://speakeasy.com/"><img src="https://custom-icon-badges.demolab.com/badge/-Built%20By%20Speakeasy-212015?style=for-the-badge&logoColor=FBE331&logo=speakeasy&labelColor=545454" /></a>
    <!-- License Badge -->
    <a href="/LICENSE"><img alt="Software License" src="https://img.shields.io/badge/license-MIT-blue.svg?style=for-the-badge"></a>    
  </p>
</p>

## Main Packages

### [arazzo](./arazzo)

The `arazzo` package provides an API for working with Arazzo documents including reading, creating, mutating, walking and validating them.

### [openapi](./openapi)

The `openapi` package provides an API for working with OpenAPI documents including reading, creating, mutating, walking, validating and upgrading them. Supports both OpenAPI 3.0.x and 3.1.x specifications.

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
