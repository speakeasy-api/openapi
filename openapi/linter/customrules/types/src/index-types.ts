import type {
  OpenAPI,
  Operation,
  Parameter,
  Response,
  RequestBody,
  Header,
  Example,
  Link,
  Callback,
  PathItem,
  SecurityScheme,
  SecurityRequirement,
  Server,
  ServerVariable,
  Tag,
  Schema,
  MediaType,
  Encoding,
  ExternalDocumentation,
  Discriminator,
} from './document';

/**
 * Index provides pre-computed collections of document elements
 * for efficient rule iteration.
 *
 * Note: All field names are lowercased when accessed from JavaScript
 * due to goja's UncapFieldNameMapper.
 */
export interface Index {
  /** Reference to the root document */
  readonly doc: OpenAPI;

  // External Documentation
  readonly externalDocumentation: IndexNode<ExternalDocumentation>[];

  // Tags
  readonly tags: IndexNode<Tag>[];

  // Servers
  readonly servers: IndexNode<Server>[];
  readonly serverVariables: IndexNode<ServerVariable>[];

  // Schemas (categorized by location)
  readonly booleanSchemas: IndexNode<Schema>[];
  readonly inlineSchemas: IndexNode<Schema>[];
  readonly componentSchemas: IndexNode<Schema>[];
  readonly externalSchemas: IndexNode<Schema>[];
  readonly schemaReferences: IndexNode<Schema>[];

  // Path Items
  readonly inlinePathItems: IndexNode<PathItem>[];
  readonly componentPathItems: IndexNode<PathItem>[];
  readonly externalPathItems: IndexNode<PathItem>[];
  readonly pathItemReferences: IndexNode<PathItem>[];

  // Operations
  readonly operations: IndexNode<Operation>[];

  // Parameters
  readonly inlineParameters: IndexNode<Parameter>[];
  readonly componentParameters: IndexNode<Parameter>[];
  readonly externalParameters: IndexNode<Parameter>[];
  readonly parameterReferences: IndexNode<Parameter>[];

  // Responses
  readonly responses: IndexNode<any>[]; // Response containers
  readonly inlineResponses: IndexNode<Response>[];
  readonly componentResponses: IndexNode<Response>[];
  readonly externalResponses: IndexNode<Response>[];
  readonly responseReferences: IndexNode<Response>[];

  // Request Bodies
  readonly inlineRequestBodies: IndexNode<RequestBody>[];
  readonly componentRequestBodies: IndexNode<RequestBody>[];
  readonly externalRequestBodies: IndexNode<RequestBody>[];
  readonly requestBodyReferences: IndexNode<RequestBody>[];

  // Headers
  readonly inlineHeaders: IndexNode<Header>[];
  readonly componentHeaders: IndexNode<Header>[];
  readonly externalHeaders: IndexNode<Header>[];
  readonly headerReferences: IndexNode<Header>[];

  // Examples
  readonly inlineExamples: IndexNode<Example>[];
  readonly componentExamples: IndexNode<Example>[];
  readonly externalExamples: IndexNode<Example>[];
  readonly exampleReferences: IndexNode<Example>[];

  // Links
  readonly inlineLinks: IndexNode<Link>[];
  readonly componentLinks: IndexNode<Link>[];
  readonly externalLinks: IndexNode<Link>[];
  readonly linkReferences: IndexNode<Link>[];

  // Callbacks
  readonly inlineCallbacks: IndexNode<Callback>[];
  readonly componentCallbacks: IndexNode<Callback>[];
  readonly externalCallbacks: IndexNode<Callback>[];
  readonly callbackReferences: IndexNode<Callback>[];

  // Security
  readonly componentSecuritySchemes: IndexNode<SecurityScheme>[];
  readonly securitySchemeReferences: IndexNode<SecurityScheme>[];
  readonly securityRequirements: IndexNode<SecurityRequirement>[];

  // Other
  readonly discriminators: IndexNode<Discriminator>[];
  readonly mediaTypes: IndexNode<MediaType>[];
  readonly encodings: IndexNode<Encoding>[];

  // Description/Summary helpers (nodes that have these fields)
  readonly descriptionNodes: IndexNode<Descriptioner>[];
  readonly summaryNodes: IndexNode<Summarizer>[];
  readonly descriptionAndSummaryNodes: IndexNode<DescriptionAndSummary>[];
}

/**
 * IndexNode wraps a document element with its location.
 */
export interface IndexNode<T> {
  /** The document element */
  readonly node: T;

  /** Location path to this element */
  readonly location: Locations;
}

/**
 * Locations is the path from root to an element.
 */
export type Locations = LocationContext[];

/**
 * LocationContext represents one level in the path.
 */
export interface LocationContext {
  /** Parent field name (e.g., "paths", "responses") */
  parentField(): string;

  /** Parent key if in a map (e.g., "/users/{id}") */
  parentKey(): string;

  /** Parent index if in an array */
  parentIndex(): number;

  /** Convert to JSON pointer string */
  toJSONPointer(): string;
}

/**
 * Interface for elements with a description.
 */
export interface Descriptioner {
  getDescription(): string;
}

/**
 * Interface for elements with a summary.
 */
export interface Summarizer {
  getSummary(): string;
}

/**
 * Interface for elements with both description and summary.
 */
export interface DescriptionAndSummary extends Descriptioner, Summarizer {}
