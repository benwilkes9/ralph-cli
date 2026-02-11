package stream

// IterationStats holds stats for a single loop iteration.
type IterationStats struct {
	PeakContext    int     // max(input + cache_creation + cache_read) across turns
	Cost           float64 // from result event
	SubagentTokens int     // sum of totalTokens from Task results
	ToolCalls      int     // number of tool invocations
}

// CumulativeStats holds stats across all iterations.
type CumulativeStats struct {
	Iterations     int
	PeakContext    int
	SubagentTokens int
	TotalCost      float64
}

// Update merges an iteration's stats into the cumulative totals.
func (c *CumulativeStats) Update(iter *IterationStats) {
	c.Iterations++
	if iter.PeakContext > c.PeakContext {
		c.PeakContext = iter.PeakContext
	}
	c.SubagentTokens += iter.SubagentTokens
	c.TotalCost += iter.Cost
}
