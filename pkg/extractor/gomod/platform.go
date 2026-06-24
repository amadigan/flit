package gomod

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/build"
	"io"
	"log"
	"os"
	"os/exec"
	"slices"
	"sort"
)

type GoPlatform struct {
	GOOS         string
	GOARCH       string
	Name         string
	CgoSupported bool
	FirstClass   bool
}

var ErrNoGo = errors.New("go executable not found in PATH")

func runGo(rv any, args ...string) (string, error) {
	goPath, err := exec.LookPath("go")
	if err != nil {
		return "", ErrNoGo
	}

	cmd := exec.Cmd{
		Args: append([]string{goPath}, args...),
		Env:  os.Environ(),
		Path: goPath,
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}

	err = json.Unmarshal(bytes.TrimSpace(out), rv)
	if err != nil {
		return string(out), err
	}

	return "", nil
}

func GetSupportedPlatforms() ([]GoPlatform, error) {
	var platforms []GoPlatform
	out, err := runGo(&platforms, "tool", "dist", "list", "-json")
	if err != nil {
		return nil, fmt.Errorf("failed to get supported platforms: %w, %s", err, out)
	}

	for i, p := range platforms {
		p.Name = p.GOOS + "/" + p.GOARCH
		platforms[i] = p
	}

	return platforms, nil
}

type PlatformTable struct {
	allPlatforms []GoPlatform
	os           map[string][]GoPlatform
	aliases      []platformAlias
}

type platformAlias struct {
	alias  string
	values []GoPlatform
}

func NewPlatformTable(platforms []GoPlatform) PlatformTable {
	pt := PlatformTable{
		allPlatforms: platforms,
		os:           make(map[string][]GoPlatform),
	}

	for _, p := range platforms {
		pt.os[p.GOOS] = append(pt.os[p.GOOS], p)
	}

	for os, ps := range pt.os {
		sorted := make([]GoPlatform, len(ps))
		copy(sorted, ps)
		sort.SliceStable(sorted, func(i, j int) bool {
			if sorted[i].FirstClass != sorted[j].FirstClass {
				return sorted[i].FirstClass
			}
			if sorted[i].CgoSupported != sorted[j].CgoSupported {
				return sorted[i].CgoSupported
			}
			return sorted[i].GOARCH < sorted[j].GOARCH
		})
		pt.os[os] = sorted
	}

	pt.os["unix"] = probeUnixPlatforms(pt.os)

	pt.aliases = []platformAlias{
		{
			alias:  "all",
			values: platforms,
		},
	}

	for osName, ps := range pt.os {
		pt.aliases = append(pt.aliases, platformAlias{
			alias:  osName,
			values: ps,
		})
	}

	sort.SliceStable(pt.aliases, func(i, j int) bool {
		return len(pt.aliases[i].values) > len(pt.aliases[j].values)
	})

	return pt
}

func probeUnixPlatforms(osMap map[string][]GoPlatform) []GoPlatform {
	unixPlatforms := []GoPlatform{}
	testFileContent := []byte("//go:build unix\n\npackage main\n")
	for osName, platforms := range osMap {
		bctx := build.Context{
			GOOS:       osName,
			GOARCH:     platforms[0].GOARCH,
			CgoEnabled: platforms[0].CgoSupported,
			OpenFile: func(path string) (io.ReadCloser, error) {
				if path == "test.go" {
					return io.NopCloser(bytes.NewReader(testFileContent)), nil
				}
				return nil, os.ErrNotExist
			},
		}

		match, err := bctx.MatchFile("", "test.go")
		if err != nil {
			log.Printf("failed to match file for os %s: %v", osName, err)
			continue
		}
		if match {
			unixPlatforms = append(unixPlatforms, platforms...)
		}
	}

	return unixPlatforms
}

func (pt PlatformTable) AllPlatforms() []GoPlatform {
	return pt.allPlatforms
}

func (pt PlatformTable) PlatformsForOS(os string) []GoPlatform {
	return pt.os[os]
}

func (pt PlatformTable) AllOSes() []string {
	oses := make([]string, 0, len(pt.os))
	for os := range pt.os {
		oses = append(oses, os)
	}
	sort.Strings(oses)
	return oses
}

func (pt PlatformTable) CollapseNames(names []string) []string {
	for _, alias := range pt.aliases {
		if len(alias.values) == 1 {
			continue
		}
		if containsAllPlatforms(alias.values, names) {
			newNames := make([]string, 0, len(names))
			for _, name := range names {
				if !containsPlatform(alias.values, name) {
					newNames = append(newNames, name)
				}
			}
			newNames = append(newNames, alias.alias)
			names = newNames
		}
	}

	return names
}

func containsAllPlatforms(platforms []GoPlatform, names []string) bool {
	if len(names) < len(platforms) {
		return false
	}

	for _, p := range platforms {
		if !slices.Contains(names, p.Name) {
			log.Printf("platform %s is not in names %v", p.Name, names)
			return false
		}
	}

	return true
}

func containsPlatform(platforms []GoPlatform, name string) bool {
	for _, p := range platforms {
		if p.Name == name {
			return true
		}
	}
	return false
}
