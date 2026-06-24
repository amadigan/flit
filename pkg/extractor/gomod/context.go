package gomod

import "go/build"

type Context struct {
	AllContext    build.Context
	BuildContexts []build.Context
	PlatformTable PlatformTable
}

func NewContext(platforms []GoPlatform) *Context {
	ctx := &Context{
		AllContext: build.Context{
			GOOS:   "all",
			GOARCH: "all",
		},
		PlatformTable: NewPlatformTable(platforms),
	}

	for _, platform := range platforms {
		ctx.BuildContexts = append(ctx.BuildContexts, build.Context{
			GOOS:       platform.GOOS,
			GOARCH:     platform.GOARCH,
			CgoEnabled: platform.CgoSupported,
		})
	}

	return ctx
}

func DefaultContext() (*Context, error) {
	platforms, err := GetSupportedPlatforms()
	if err != nil {
		return nil, err
	}

	return NewContext(platforms), nil
}
