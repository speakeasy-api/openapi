import type { Index } from './index-types';

/**
 * DocumentInfo provides access to the OpenAPI document and its index.
 *
 * Note: All field and method names are lowercased when accessed from JavaScript
 * due to goja's UncapFieldNameMapper.
 */
export interface DocumentInfo {
  /** The parsed OpenAPI document */
  readonly document: OpenAPI;

  /** Absolute path or URL of the document */
  readonly location: string;

  /** Pre-computed index for efficient document traversal */
  readonly index: Index;
}

/**
 * OpenAPI document root object.
 * Access nested elements via fields or getter methods.
 */
export interface OpenAPI {
  /** OpenAPI version (e.g., "3.1.0") */
  readonly openAPI: string;

  /** Document info section */
  getInfo(): Info | null;

  /** Document paths */
  getPaths(): Paths | null;

  /** Document components */
  getComponents(): Components | null;

  /** Document servers */
  getServers(): Server[] | null;

  /** Document tags */
  getTags(): Tag[] | null;

  /** Document security requirements */
  getSecurity(): SecurityRequirement[] | null;

  /** Get the root YAML node */
  getRootNode(): any;
}

export interface Info {
  readonly title: string;
  readonly version: string;
  getDescription(): string;
  getSummary(): string;
  getTermsOfService(): string;
  getContact(): Contact | null;
  getLicense(): License | null;
  getRootNode(): any;
}

export interface Contact {
  getName(): string;
  getURL(): string;
  getEmail(): string;
  getRootNode(): any;
}

export interface License {
  readonly name: string;
  getIdentifier(): string;
  getURL(): string;
  getRootNode(): any;
}

export interface Paths {
  /** Get all path item entries */
  all(): IterableIterator<[string, PathItem]>;
  getRootNode(): any;
}

export interface PathItem {
  getSummary(): string;
  getDescription(): string;
  getParameters(): Parameter[];
  getServers(): Server[];

  /** Get a specific operation by HTTP method */
  getOperation(method: string): Operation | null;

  /** Get all operations */
  all(): IterableIterator<[string, Operation]>;

  getRootNode(): any;
}

export interface Operation {
  getOperationID(): string;
  getSummary(): string;
  getDescription(): string;
  getTags(): string[];
  getParameters(): Parameter[];
  getRequestBody(): RequestBody | null;
  getResponses(): Responses | null;
  getSecurity(): SecurityRequirement[] | null;
  getServers(): Server[];
  getDeprecated(): boolean;
  getRootNode(): any;
}

export interface Parameter {
  readonly name: string;
  readonly in: string;
  getDescription(): string;
  getRequired(): boolean;
  getDeprecated(): boolean;
  getSchema(): Schema | null;
  getRootNode(): any;
}

export interface RequestBody {
  getDescription(): string;
  getRequired(): boolean;
  getContent(): Map<string, MediaType>;
  getRootNode(): any;
}

export interface Responses {
  /** Get all response entries */
  all(): IterableIterator<[string, Response]>;
  getDefault(): Response | null;
  getRootNode(): any;
}

export interface Response {
  readonly description: string;
  getHeaders(): Map<string, Header>;
  getContent(): Map<string, MediaType>;
  getLinks(): Map<string, Link>;
  getRootNode(): any;
}

export interface Header {
  getDescription(): string;
  getRequired(): boolean;
  getDeprecated(): boolean;
  getSchema(): Schema | null;
  getRootNode(): any;
}

export interface MediaType {
  getSchema(): Schema | null;
  getExample(): any;
  getExamples(): Map<string, Example>;
  getEncoding(): Map<string, Encoding>;
  getRootNode(): any;
}

export interface Encoding {
  getContentType(): string;
  getHeaders(): Map<string, Header>;
  getStyle(): string;
  getExplode(): boolean;
  getAllowReserved(): boolean;
  getRootNode(): any;
}

export interface Example {
  getSummary(): string;
  getDescription(): string;
  getValue(): any;
  getExternalValue(): string;
  getRootNode(): any;
}

export interface Link {
  getOperationID(): string;
  getOperationRef(): string;
  getDescription(): string;
  getServer(): Server | null;
  getRootNode(): any;
}

export interface Components {
  getSchemas(): Map<string, Schema>;
  getResponses(): Map<string, Response>;
  getParameters(): Map<string, Parameter>;
  getRequestBodies(): Map<string, RequestBody>;
  getHeaders(): Map<string, Header>;
  getSecuritySchemes(): Map<string, SecurityScheme>;
  getLinks(): Map<string, Link>;
  getCallbacks(): Map<string, Callback>;
  getPathItems(): Map<string, PathItem>;
  getRootNode(): any;
}

export interface Schema {
  getType(): string | string[];
  getFormat(): string;
  getTitle(): string;
  getDescription(): string;
  getDefault(): any;
  getEnum(): any[];
  getRequired(): string[];
  getProperties(): Map<string, Schema>;
  getItems(): Schema | null;
  getAdditionalProperties(): Schema | boolean | null;
  getMinimum(): number | null;
  getMaximum(): number | null;
  getMinLength(): number | null;
  getMaxLength(): number | null;
  getMinItems(): number | null;
  getMaxItems(): number | null;
  getPattern(): string;
  getNullable(): boolean;
  getDeprecated(): boolean;
  getAllOf(): Schema[];
  getAnyOf(): Schema[];
  getOneOf(): Schema[];
  getNot(): Schema | null;
  getRootNode(): any;
}

export interface Server {
  readonly url: string;
  getDescription(): string;
  getVariables(): Map<string, ServerVariable>;
  getRootNode(): any;
}

export interface ServerVariable {
  readonly default: string;
  getEnum(): string[];
  getDescription(): string;
  getRootNode(): any;
}

export interface Tag {
  readonly name: string;
  getDescription(): string;
  getExternalDocs(): ExternalDocumentation | null;
  getRootNode(): any;
}

export interface ExternalDocumentation {
  readonly url: string;
  getDescription(): string;
  getRootNode(): any;
}

export interface SecurityScheme {
  readonly type: string;
  getDescription(): string;
  getName(): string;
  getIn(): string;
  getScheme(): string;
  getBearerFormat(): string;
  getFlows(): OAuthFlows | null;
  getOpenIdConnectUrl(): string;
  getRootNode(): any;
}

export interface OAuthFlows {
  getImplicit(): OAuthFlow | null;
  getPassword(): OAuthFlow | null;
  getClientCredentials(): OAuthFlow | null;
  getAuthorizationCode(): OAuthFlow | null;
  getRootNode(): any;
}

export interface OAuthFlow {
  getAuthorizationUrl(): string;
  getTokenUrl(): string;
  getRefreshUrl(): string;
  getScopes(): Map<string, string>;
  getRootNode(): any;
}

export interface SecurityRequirement {
  /** Get all security requirement entries */
  all(): IterableIterator<[string, string[]]>;
  getRootNode(): any;
}

export interface Callback {
  /** Get all callback entries */
  all(): IterableIterator<[string, PathItem]>;
  getRootNode(): any;
}

export interface Discriminator {
  readonly propertyName: string;
  getMapping(): Map<string, string>;
  getRootNode(): any;
}
