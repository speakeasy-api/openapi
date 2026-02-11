package rules

// FixAvailable returns true for rules that provide auto-fix suggestions.
// This satisfies the linter.DocumentedRule interface's FixAvailable() method.

func (r *PathTrailingSlashRule) FixAvailable() bool                    { return true }
func (r *OAS3HostTrailingSlashRule) FixAvailable() bool                { return true }
func (r *OwaspSecurityHostsHttpsOAS3Rule) FixAvailable() bool          { return true }
func (r *DuplicatedEnumRule) FixAvailable() bool                       { return true }
func (r *OAS3NoNullableRule) FixAvailable() bool                       { return true }
func (r *OperationTagDefinedRule) FixAvailable() bool                  { return true }
func (r *TagsAlphabeticalRule) FixAvailable() bool                     { return true }
func (r *OwaspJWTBestPracticesRule) FixAvailable() bool                { return true }
func (r *OwaspNoAdditionalPropertiesRule) FixAvailable() bool          { return true }
func (r *OwaspDefineErrorResponses401Rule) FixAvailable() bool         { return true }
func (r *OwaspDefineErrorResponses429Rule) FixAvailable() bool         { return true }
func (r *OwaspDefineErrorResponses500Rule) FixAvailable() bool         { return true }
func (r *OwaspDefineErrorValidationRule) FixAvailable() bool           { return true }
func (r *OwaspRateLimitRetryAfterRule) FixAvailable() bool             { return true }
func (r *InfoDescriptionRule) FixAvailable() bool                      { return true }
func (r *InfoContactRule) FixAvailable() bool                          { return true }
func (r *InfoLicenseRule) FixAvailable() bool                          { return true }
func (r *LicenseURLRule) FixAvailable() bool                           { return true }
func (r *ComponentDescriptionRule) FixAvailable() bool                 { return true }
func (r *TagDescriptionRule) FixAvailable() bool                       { return true }
func (r *OperationDescriptionRule) FixAvailable() bool                 { return true }
func (r *OAS3ParameterDescriptionRule) FixAvailable() bool             { return true }
func (r *OperationTagsRule) FixAvailable() bool                        { return true }
func (r *ContactPropertiesRule) FixAvailable() bool                    { return true }
func (r *OAS3HostNotExampleRule) FixAvailable() bool                   { return true }
func (r *OAS3APIServersRule) FixAvailable() bool                       { return true }
func (r *OwaspIntegerFormatRule) FixAvailable() bool                   { return true }
func (r *OwaspStringLimitRule) FixAvailable() bool                     { return true }
func (r *OwaspArrayLimitRule) FixAvailable() bool                      { return true }
func (r *OwaspIntegerLimitRule) FixAvailable() bool                    { return true }
func (r *OwaspAdditionalPropertiesConstrainedRule) FixAvailable() bool { return true }
func (r *UnusedComponentRule) FixAvailable() bool                      { return true }
func (r *PathParamsRule) FixAvailable() bool                           { return true }
