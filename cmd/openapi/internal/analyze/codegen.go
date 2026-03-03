package analyze

import (
	"fmt"
	"slices"
)

// CodegenTier represents the difficulty level for code generation.
type CodegenTier int

const (
	// CodegenGreen means the schema is straightforward to generate code for.
	CodegenGreen CodegenTier = iota
	// CodegenYellow means the schema has moderate complexity that may challenge some generators.
	CodegenYellow
	// CodegenRed means the schema has significant challenges for code generation.
	CodegenRed
)

func (t CodegenTier) String() string {
	switch t {
	case CodegenGreen:
		return "green"
	case CodegenYellow:
		return "yellow"
	case CodegenRed:
		return "red"
	}
	return "unknown"
}

// CodegenSignal describes a specific code generation challenge.
type CodegenSignal struct {
	// ID is a short identifier for the signal.
	ID string
	// Description is a human-readable explanation.
	Description string
	// Severity is the impact level.
	Severity CodegenTier
	// AffectedSchemas lists the schema IDs affected (if applicable).
	AffectedSchemas []string
}

// CodegenDifficulty holds the codegen assessment for a single schema.
type CodegenDifficulty struct {
	SchemaID string
	Tier     CodegenTier
	Signals  []*CodegenSignal
}

// CodegenReport is the aggregate codegen assessment for the entire document.
type CodegenReport struct {
	// PerSchema maps schema ID to its difficulty assessment.
	PerSchema map[string]*CodegenDifficulty
	// GreenCount is the number of green-tier schemas.
	GreenCount int
	// YellowCount is the number of yellow-tier schemas.
	YellowCount int
	// RedCount is the number of red-tier schemas.
	RedCount int
	// CompatibilityScore is the percentage of schemas that are green-tier (0-100).
	CompatibilityScore float64
	// TopSignals are the most impactful signals across all schemas.
	TopSignals []*CodegenSignal
}

// AssessCodegen evaluates code generation difficulty for all schemas.
func AssessCodegen(g *Graph, cycles *CycleAnalysis, metrics map[string]*SchemaMetrics) *CodegenReport {
	report := &CodegenReport{
		PerSchema: make(map[string]*CodegenDifficulty, len(g.Nodes)),
	}

	signalCounts := make(map[string]int)

	for id, node := range g.Nodes {
		m := metrics[id]
		d := &CodegenDifficulty{
			SchemaID: id,
			Tier:     CodegenGreen,
		}

		// Required cycle membership
		if m != nil && m.InSCC {
			for _, cycle := range cycles.Cycles {
				if cycle.HasRequiredOnlyPath && slices.Contains(cycle.Path, id) {
					d.addSignal("required-cycle", "Part of a cycle where all edges are required — many languages need pointer/boxing types", CodegenRed)
					signalCounts["required-cycle"]++
					break
				}
			}
			if len(d.Signals) == 0 {
				d.addSignal("optional-cycle", "Part of a cycle but has optional/nullable break points", CodegenYellow)
				signalCounts["optional-cycle"]++
			}
		}

		// Aggregate union site info for deduplicated signals
		var undiscriminatedOneOfs, anyOfSites, largeUnions []UnionSite
		for _, site := range node.UnionSites {
			if site.Kind == "oneOf" && !site.HasDiscriminator {
				undiscriminatedOneOfs = append(undiscriminatedOneOfs, site)
			}
			if site.Kind == "anyOf" {
				anyOfSites = append(anyOfSites, site)
			}
			if site.Width > 5 {
				largeUnions = append(largeUnions, site)
			}
		}

		if len(undiscriminatedOneOfs) == 1 {
			site := undiscriminatedOneOfs[0]
			d.addSignal("oneOf-no-discriminator",
				fmt.Sprintf("oneOf at %s without discriminator — codegen must trial-deserialize (%d variants)", site.Path, site.Width),
				CodegenYellow)
			signalCounts["oneOf-no-discriminator"]++
		} else if len(undiscriminatedOneOfs) > 1 {
			d.addSignal("oneOf-no-discriminator",
				fmt.Sprintf("%d oneOf sites without discriminator — codegen must trial-deserialize", len(undiscriminatedOneOfs)),
				CodegenYellow)
			signalCounts["oneOf-no-discriminator"]++
		}

		if len(anyOfSites) == 1 {
			site := anyOfSites[0]
			d.addSignal("anyOf",
				fmt.Sprintf("anyOf at %s with %d potentially overlapping shapes — hard to generate correct types", site.Path, site.Width),
				CodegenRed)
			signalCounts["anyOf"]++
		} else if len(anyOfSites) > 1 {
			d.addSignal("anyOf",
				fmt.Sprintf("%d anyOf sites with potentially overlapping shapes — hard to generate correct types", len(anyOfSites)),
				CodegenRed)
			signalCounts["anyOf"]++
		}

		if len(largeUnions) == 1 {
			site := largeUnions[0]
			d.addSignal("large-union",
				fmt.Sprintf("%s at %s has %d variants — large type unions are expensive to generate", site.Kind, site.Path, site.Width),
				CodegenYellow)
			signalCounts["large-union"]++
		} else if len(largeUnions) > 1 {
			maxW := 0
			for _, site := range largeUnions {
				if site.Width > maxW {
					maxW = site.Width
				}
			}
			d.addSignal("large-union",
				fmt.Sprintf("%d union sites with >5 variants (largest: %d)", len(largeUnions), maxW),
				CodegenYellow)
			signalCounts["large-union"]++
		}

		// Combinatorial explosion across multiple independent union sites
		if len(node.UnionSites) > 1 {
			vp := 1
			for _, site := range node.UnionSites {
				vp *= site.Width
			}
			if vp > 20 {
				severity := CodegenYellow
				if vp > 100 {
					severity = CodegenRed
				}
				d.addSignal("combinatorial-explosion",
					fmt.Sprintf("%d union sites produce %d variant combinations", len(node.UnionSites), vp),
					severity)
				signalCounts["combinatorial-explosion"]++
			}
		}

		// Mixed types
		if len(node.Types) > 1 {
			hasNull := slices.Contains(node.Types, "null")
			nonNullTypes := 0
			for _, t := range node.Types {
				if t != "null" {
					nonNullTypes++
				}
			}
			if nonNullTypes > 1 {
				d.addSignal("mixed-types", "Multiple non-null types — type unions not expressible in many languages", CodegenRed)
				signalCounts["mixed-types"]++
			} else if hasNull && nonNullTypes == 1 {
				// Just nullable — this is fine for most languages
			}
		}

		// additionalProperties with named properties
		hasAdditionalProps := false
		for _, e := range g.OutEdges[id] {
			if e.Kind == EdgeAdditionalProperties {
				hasAdditionalProps = true
				break
			}
		}
		if hasAdditionalProps && node.PropertyCount > 0 {
			d.addSignal("mixed-map-struct", "additionalProperties combined with named properties — awkward map+struct hybrid", CodegenYellow)
			signalCounts["mixed-map-struct"]++
		}

		// Deep allOf chains
		allOfEdges := 0
		for _, e := range g.OutEdges[id] {
			if e.Kind == EdgeAllOf {
				allOfEdges++
			}
		}
		if allOfEdges > 2 {
			d.addSignal("deep-allOf", "Deep allOf composition — may cause inheritance complexity or flattening issues", CodegenYellow)
			signalCounts["deep-allOf"]++
		}

		// Very high property count
		if node.PropertyCount > 30 {
			d.addSignal("high-property-count", "Schema has many properties (>30) — may indicate it should be split", CodegenYellow)
			signalCounts["high-property-count"]++
		}

		report.PerSchema[id] = d
	}

	// Aggregate counts
	for _, d := range report.PerSchema {
		switch d.Tier {
		case CodegenGreen:
			report.GreenCount++
		case CodegenYellow:
			report.YellowCount++
		case CodegenRed:
			report.RedCount++
		}
	}

	total := len(report.PerSchema)
	if total > 0 {
		report.CompatibilityScore = float64(report.GreenCount) / float64(total) * 100
	}

	// Build top signals
	for signalID, count := range signalCounts {
		report.TopSignals = append(report.TopSignals, &CodegenSignal{
			ID:          signalID,
			Description: signalID, // will be overwritten below
			AffectedSchemas: func() []string {
				var schemas []string
				for id, d := range report.PerSchema {
					for _, s := range d.Signals {
						if s.ID == signalID {
							schemas = append(schemas, id)
							break
						}
					}
				}
				return schemas
			}(),
		})
		_ = count
	}

	return report
}

func (d *CodegenDifficulty) addSignal(id, description string, severity CodegenTier) {
	d.Signals = append(d.Signals, &CodegenSignal{
		ID:          id,
		Description: description,
		Severity:    severity,
	})
	if severity > d.Tier {
		d.Tier = severity
	}
}
