package docker

type fakeRunner struct {
	calls  [][]string
	errFor map[string]error
}

func (f *fakeRunner) Run(name string, args ...string) error {
	call := make([]string, 0, 1+len(args))
	call = append(call, name)
	call = append(call, args...)
	f.calls = append(f.calls, call)
	return f.errFor[name]
}
