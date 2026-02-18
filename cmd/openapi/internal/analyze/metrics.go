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

func topSchemasBy(metrics map[string]*SchemaMetrics, n int, less func(a, b *SchemaMetrics) bool) []*SchemaMetrics {
	all := make([]*SchemaMetrics, 0, len(metrics))
	for _, m := range metrics {
		all = append(all, m)
	}
	sort.Slice(all, func(i, j int) bool {
		return less(all[i], all[j])
	})
	if n > len(all) {
		n = len(all)
	}
	return all[:n]
}
