/**
 * @speakeasy-api/openapi-linter-types
 *
 * TypeScript types for writing custom OpenAPI linter rules.
 *
 * @example
 * ```typescript
 * import { Rule, registerRule, createValidationError } from '@speakeasy-api/openapi-linter-types';
 * import type { Context, DocumentInfo, RuleConfig, Severity, ValidationError } from '@speakeasy-api/openapi-linter-types';
 *
 * class RequireOperationSummary extends Rule {
 *   id(): string { return 'custom-require-operation-summary'; }
 *   category(): string { return 'style'; }
 *   description(): string { return 'All operations must have a summary.'; }
 *   summary(): string { return 'Operations must have summary'; }
 *
 *   run(ctx: Context, docInfo: DocumentInfo, config: RuleConfig): ValidationError[] {
 *     const errors: ValidationError[] = [];
 *
 *     for (const opNode of docInfo.index.operations) {
 *       const op = opNode.node;
 *       if (!op.getSummary()) {
 *         errors.push(createValidationError(
 *           config.getSeverity(this.defaultSeverity()),
 *           this.id(),
 *           'Operation is missing a summary',
 *           op.getRootNode()
 *         ));
 *       }
 *     }
 *
 *     return errors;
 *   }
 * }
 *
 * registerRule(new RequireOperationSummary());
 * ```
 */

// Core types
export type { Severity } from './severity';
export type { Context } from './context';
export type { RuleConfig } from './config';

// Rule interface and base class
export { Rule, registerRule } from './rule';
export type { RuleRunner } from './rule';

// Validation errors
export { createValidationError } from './validation';
export type { ValidationError, YamlNode } from './validation';

// Document types
export type { DocumentInfo } from './document';
export type {
  OpenAPI,
  Info,
  Contact,
  License,
  Paths,
  PathItem,
  Operation,
  Parameter,
  RequestBody,
  Responses,
  Response,
  Header,
  MediaType,
  Encoding,
  Example,
  Link,
  Components,
  Schema,
  Server,
  ServerVariable,
  Tag,
  ExternalDocumentation,
  SecurityScheme,
  OAuthFlows,
  OAuthFlow,
  SecurityRequirement,
  Callback,
  Discriminator,
} from './document';

// Index types
export type {
  Index,
  IndexNode,
  Locations,
  LocationContext,
  Descriptioner,
  Summarizer,
  DescriptionAndSummary,
} from './index-types';

// Global runtime types (console, etc.)
// Import for side effects to register global types
import './globals';
export type { Console } from './globals';
