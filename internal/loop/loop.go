package loop

// Mode represents the loop mode (plan or build).
type Mode string

// Loop modes.
const (
	ModePlan  Mode = "plan"
	ModeBuild Mode = "build"
)

// Options configures a loop run.
type Options struct {
	Mode          Mode
	PromptFile    string
	MaxIterations int
	FreshContext  bool
	LogsDir       string
	Branch        string
}
