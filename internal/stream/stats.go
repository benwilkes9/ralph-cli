package stream

// IterationStats holds stats for a single loop iteration.
type IterationStats struct {
	PeakContext    int     // max(input + cache_creation + cache_read) across turns
	Cost           float64 // from result event
	SubagentTokens int     // sum of totalTokens from Task results
	ToolCalls      int     // number of tool invocations
}

// ObserveAssistant tracks peak context from an assistant event's usage.
func (s *IterationStats) ObserveAssistant(u *Usage) {
	if u == nil {
		return
	}
	ctx := u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
	if ctx > s.PeakContext {
		s.PeakContext = ctx
	}
}

// ObserveToolUse increments the tool call counter.
func (s *IterationStats) ObserveToolUse() {
	s.ToolCalls++
}

// ObserveSubagent accumulates subagent tokens.
func (s *IterationStats) ObserveSubagent(totalTokens int) {
	s.SubagentTokens += totalTokens
}

// ObserveResult records the iteration cost.
func (s *IterationStats) ObserveResult(costUSD float64) {
	s.Cost = costUSD
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
