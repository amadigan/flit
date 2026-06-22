package gomod

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
)

type ResolvedModule struct {
	Path    string
	Version string
	Zip     string
}

type ResolverError struct {
	Message string
	Cause   error
}

func (e *ResolverError) Error() string {
	return e.Message + ": " + e.Cause.Error()
}

func (e *ResolverError) Unwrap() error {
	return e.Cause
}

func Resolve(name, version string) (ResolvedModule, error) {
	if version == "" {
		version = "latest"
	}

	cmd := exec.Cmd{
		Args: []string{"go", "mod", "download", "-json", name + "@" + version},
		Env:  os.Environ(),
	}

	if path, err := exec.LookPath("go"); err == nil {
		cmd.Path = path
	} else {
		return ResolvedModule{}, &ResolverError{Message: "go executable not found in PATH", Cause: err}
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return ResolvedModule{}, &ResolverError{Message: string(out), Cause: err}
	}

	var mod ResolvedModule
	err = json.Unmarshal(bytes.TrimSpace(out), &mod)
	if err != nil {
		return ResolvedModule{}, &ResolverError{Message: "failed to unmarshal JSON", Cause: err}
	}

	return mod, nil
}
