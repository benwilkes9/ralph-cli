package loop

// DefaultMaxStale is the number of consecutive stale iterations before aborting.
const DefaultMaxStale = 2

// StaleDetector tracks consecutive iterations with no new commits.
type StaleDetector struct {
	maxStale   int
	staleCount int
	lastHead   string
}

// NewStaleDetector creates a detector that triggers after maxStale consecutive stale iterations.
func NewStaleDetector(maxStale int) *StaleDetector {
	if maxStale <= 0 {
		maxStale = DefaultMaxStale
	}
	return &StaleDetector{maxStale: maxStale}
}

// Check compares the current HEAD to the previous. Returns true if the loop should abort.
func (d *StaleDetector) Check(currentHead string) (abort bool, staleCount int) {
	if d.lastHead == "" {
		d.lastHead = currentHead
		return false, 0
	}

	if currentHead == d.lastHead {
		d.staleCount++
	} else {
		d.staleCount = 0
	}

	d.lastHead = currentHead
	return d.staleCount >= d.maxStale, d.staleCount
}

// MaxStale returns the configured threshold.
func (d *StaleDetector) MaxStale() int {
	return d.maxStale
}
