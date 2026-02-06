// Auto-generated from Spectral rule: version-semver
// Original description: Version must be semver format
import {
  Rule,
  registerRule,
  createValidationError,
} from '@speakeasy-api/openapi-linter-types';
import type {
  Context,
  DocumentInfo,
  RuleConfig,
  ValidationError,
} from '@speakeasy-api/openapi-linter-types';

class VersionSemver extends Rule {
  id(): string { return 'custom-version-semver'; }
  category(): string { return 'style'; }
  description(): string { return 'Version must be semver format'; }
  summary(): string { return 'Version must be semver format'; }
  defaultSeverity(): 'error' | 'warning' | 'hint' { return 'error'; }

  run(_ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
    const errors: ValidationError[] = [];
    const target = docInfo.document.getInfo();
    if (target) {
      // WARNING: No known getter for field 'version' — using dynamic access
if (!('version' in (target as any))) { /* field may not exist */ }
const value = (target as any)['version'];
      {
        let re: RegExp | null = null;
        try {
          re = new RegExp('^[0-9]+\\.[0-9]+\\.[0-9]+');
        } catch (e) {
          errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), `Invalid regex pattern: ${e}`, docInfo.document.getRootNode()));
        }
        if (re && !re.test(String(value || ''))) {
          errors.push(createValidationError(config.getSeverity(this.defaultSeverity()), this.id(), 'Version must be semver format', target.getRootNode ? target.getRootNode() : null));
        }
      }
    }
    return errors;
  }
}

registerRule(new VersionSemver());
