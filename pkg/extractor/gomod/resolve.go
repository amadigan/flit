package gomod

import (
	"archive/zip"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"time"
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
	fullName := name
	if !strings.Contains(name, "@") {
		if version == "" {
			version = "latest"
		}
		fullName = name + "@" + version
	}

	var mod ResolvedModule
	out, err := runGo(&mod, "mod", "download", "-json", fullName)
	if err != nil {
		return ResolvedModule{}, &ResolverError{Message: out, Cause: err}
	}

	return mod, nil
}

type ModuleSource struct {
	file          *os.File
	zip           *zip.Reader
	prefix        string
	entries       map[string]ModuleFileEntry
	platformTable PlatformTable
	module        ResolvedModule
}

func OpenModuleSource(platformTable PlatformTable, mod ResolvedModule) (*ModuleSource, error) {
	f, err := os.Open(mod.Zip)
	if err != nil {
		return nil, err
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	zr, err := zip.NewReader(f, fi.Size())
	if err != nil {
		f.Close()
		return nil, err
	}

	ms := &ModuleSource{
		file:          f,
		zip:           zr,
		prefix:        mod.Path + "@" + mod.Version + "/",
		entries:       make(map[string]ModuleFileEntry),
		platformTable: platformTable,
		module:        mod,
	}

	for _, entry := range zr.File {
		if after, ok := strings.CutPrefix(entry.Name, ms.prefix); ok {
			entryPath, dir := strings.CutSuffix(after, "/")
			name := path.Base(entryPath)
			ms.entries[entryPath] = ModuleFileEntry{
				entry: entry,
				path:  entryPath,
				name:  name,
				dir:   dir,
			}
		}
	}

	return ms, nil
}

func (ms *ModuleSource) Module() ResolvedModule {
	return ms.module
}

func (ms *ModuleSource) Close() error {
	return ms.file.Close()
}

func (ms *ModuleSource) Open(name string) (io.ReadCloser, error) {
	entry, ok := ms.entries[name]
	if !ok {
		return nil, os.ErrNotExist
	}
	return entry.Open()
}

func (ms *ModuleSource) ListEntries() []ModuleFileEntry {
	entries := make([]ModuleFileEntry, 0, len(ms.entries))
	for _, entry := range ms.entries {
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].path < entries[j].path
	})

	return entries
}

type ModuleFileEntry struct {
	entry *zip.File
	path  string
	name  string
	dir   bool
}

var _ os.FileInfo = ModuleFileEntry{}

func (e ModuleFileEntry) Path() string {
	return e.path
}

func (e ModuleFileEntry) Name() string {
	return e.name
}

func (e ModuleFileEntry) Size() int64 {
	return int64(e.entry.UncompressedSize64)
}

func (e ModuleFileEntry) Mode() os.FileMode {
	if e.dir {
		return os.ModeDir | 0755
	}
	return 0644
}

func (e ModuleFileEntry) ModTime() time.Time {
	return e.entry.Modified
}

func (e ModuleFileEntry) IsDir() bool {
	return e.dir
}

func (e ModuleFileEntry) Sys() any {
	return e.entry
}

func (e ModuleFileEntry) Open() (io.ReadCloser, error) {
	return e.entry.Open()
}
