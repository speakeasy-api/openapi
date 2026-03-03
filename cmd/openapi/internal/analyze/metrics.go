package analyze

import "sort"

// SchemaMetrics holds computed complexity metrics for a single schema.
type SchemaMetrics struct {
	// NodeID is the schema identifier.
	NodeID string
	// FanIn is the number of distinct schemas referencing this one.
	FanIn int
	// FanOut is the number of distinct schemas this one references.
	FanOut int
	// PropertyCount is the number of properties defined.
	PropertyCount int
	// RequiredCount is the number of required properties.
	RequiredCount int
	// NestingDepth is the maximum depth of inline object nesting (not counting $refs).
	NestingDepth int
	// CompositionDepth is the depth of allOf/anyOf/oneOf nesting.
	CompositionDepth int
	// HasDiscriminator is true if a discriminator is defined.
	HasDiscriminator bool
	// CycleMembership is the number of cycles this schema participates in.
	CycleMembership int
	// InSCC is true if this schema is part of a non-trivial SCC.
	InSCC bool
	// Types is the list of declared types.
	Types []string
	// DeepPropertyCount is total properties across all inline sub-schemas.
	DeepPropertyCount int
	// MaxUnionWidth is the largest oneOf/anyOf width in the schema tree.
	MaxUnionWidth int
	// VariantProduct is the product of all union widths (cross-product explosion measure).
	VariantProduct int
	// UnionSiteCount is the number of oneOf/anyOf sites in the tree.
	UnionSiteCount int
}

// ComputeMetrics calculates per-schema complexity metrics using the graph and cycle analysis.
func ComputeMetrics(g *Graph, cycles *CycleAnalysis) map[string]*SchemaMetrics {
	metrics := make(map[string]*SchemaMetrics, len(g.Nodes))

	for id, node := range g.Nodes {
		m := &SchemaMetrics{
			NodeID:           id,
			FanIn:            g.FanIn(id),
			FanOut:           g.FanOut(id),
			PropertyCount:    node.PropertyCount,
			RequiredCount:    node.RequiredCount,
			HasDiscriminator: node.HasDiscriminator,
			InSCC:            cycles.NodesInCycles[id],
			Types:            node.Types,
		}

		// Count cycle membership
		for _, cycle := range cycles.Cycles {
			for _, cid := range cycle.Path {
				if cid == id {
					m.CycleMembership++
					break
				}
			}
		}

		m.NestingDepth = node.NestingDepth
		m.CompositionDepth = node.CompositionDepth
		m.DeepPropertyCount = node.DeepPropertyCount
		m.UnionSiteCount = len(node.UnionSites)

		// Compute union metrics
		if len(node.UnionSites) > 0 {
			m.VariantProduct = 1
			for _, site := range node.UnionSites {
				if site.Width > m.MaxUnionWidth {
					m.MaxUnionWidth = site.Width
				}
				m.VariantProduct *= site.Width
			}
		}

		metrics[id] = m
	}

	return metrics
}

// TopSchemasByFanIn returns the top N schemas sorted by fan-in (most referenced first).
func TopSchemasByFanIn(metrics map[string]*SchemaMetrics, n int) []*SchemaMetrics {
	return topSchemasBy(metrics, n, func(a, b *SchemaMetrics) bool {
		return a.FanIn > b.FanIn
	})
}

// TopSchemasByFanOut returns the top N schemas sorted by fan-out (most dependencies first).
func TopSchemasByFanOut(metrics map[string]*SchemaMetrics, n int) []*SchemaMetrics {
	return topSchemasBy(metrics, n, func(a, b *SchemaMetrics) bool {
		return a.FanOut > b.FanOut
	})
}

// TopSchemasByComplexity returns the top N schemas by a composite complexity score.
func TopSchemasByComplexity(metrics map[string]*SchemaMetrics, n int) []*SchemaMetrics {
	return topSchemasBy(metrics, n, func(a, b *SchemaMetrics) bool {
		return a.ComplexityScore() > b.ComplexityScore()
	})
}

// TopSchemasByName returns all schemas sorted alphabetically by name.
func TopSchemasByName(metrics map[string]*SchemaMetrics, n int) []*SchemaMetrics {
	return topSchemasBy(metrics, n, func(a, b *SchemaMetrics) bool {
		return a.NodeID < b.NodeID
	})
}

// TopSchemasByTier returns schemas sorted by codegen tier (red first, then yellow, then green).
func TopSchemasByTier(metrics map[string]*SchemaMetrics, codegen *CodegenReport, n int) []*SchemaMetrics {
	return topSchemasBy(metrics, n, func(a, b *SchemaMetrics) bool {
		tierA := CodegenGreen
		if d, ok := codegen.PerSchema[a.NodeID]; ok {
			tierA = d.Tier
		}
		tierB := CodegenGreen
		if d, ok := codegen.PerSchema[b.NodeID]; ok {
			tierB = d.Tier
		}
		return tierA > tierB // red (2) > yellow (1) > green (0)
	})
}

// ComplexityScore returns a composite complexity score for this schema.
func (m *SchemaMetrics) ComplexityScore() int {
	score := m.FanIn + m.FanOut + m.DeepPropertyCount + m.CompositionDepth*3 + m.NestingDepth*2
	if m.InSCC {
		score += 10
	}
	score += m.CycleMembership * 5
	if m.VariantProduct > 1 {
		// Log-scale contribution of cross-product explosion
		vp := m.VariantProduct
		logContrib := 0
		for vp > 1 {
			logContrib++
			vp /= 2
		}
		score += logContrib * 5
	}
	// Multi-site bonus: independent unions multiply codegen difficulty
	if m.UnionSiteCount > 1 {
		score += m.UnionSiteCount * 3
	}
	return score
}

// ComplexityBreakdown returns a map of component names to their contribution to the complexity score.
func (m *SchemaMetrics) ComplexityBreakdown() []ScoreComponent {
	var components []ScoreComponent
	add := func(name string, value int) {
		if value > 0 {
			components = append(components, ScoreComponent{Name: name, Value: value})
		}
	}
	add("fan-in", m.FanIn)
	add("fan-out", m.FanOut)
	add("properties", m.DeepPropertyCount)
	add("composition", m.CompositionDepth*3)
	add("nesting", m.NestingDepth*2)
	if m.InSCC {
		add("in-SCC", 10)
	}
	add("cycle-membership", m.CycleMembership*5)
	if m.VariantProduct > 1 {
		vp := m.VariantProduct
		logContrib := 0
		for vp > 1 {
			logContrib++
			vp /= 2
		}
		add("variant-explosion", logContrib*5)
	}
	if m.UnionSiteCount > 1 {
		add("multi-union", m.UnionSiteCount*3)
	}
	return components
}

// ScoreComponent is a named contribution to the complexity score.
type ScoreComponent struct {
	Name  string
	Value int
}

func topSchemasBy(metrics map[string]*SchemaMetrics, n int, less func(a, b *SchemaMetrics) bool) []*SchemaMetrics {
	all := make([]*SchemaMetrics, 0, len(metrics))
	for _, m := range metrics {
		all = append(all, m)
	}
	sort.SliceStable(all, func(i, j int) bool {
		if less(all[i], all[j]) {
			return true
		}
		if less(all[j], all[i]) {
			return false
		}
		// Deterministic tie-break by name
		return all[i].NodeID < all[j].NodeID
	})
	if n > len(all) {
		n = len(all)
	}
	return all[:n]
}
