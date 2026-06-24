package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/amadigan/flit/pkg/extractor/gomod"
)

func main() {
	platforms, err := gomod.GetSupportedPlatforms()
	if err != nil {
		log.Fatalf("failed to get supported platforms: %v", err)
	}

	pt := gomod.NewPlatformTable(platforms)

	fmt.Printf("all: %s\n", joinPlatforms(pt.AllPlatforms()))

	for _, os := range pt.AllOSes() {
		fmt.Printf("%s: %s\n", os, joinPlatforms(pt.PlatformsForOS(os)))
	}

}

func joinPlatforms(platforms []gomod.GoPlatform) string {
	names := make([]string, len(platforms))
	for i, p := range platforms {
		names[i] = p.Name
	}
	return strings.Join(names, " ")
}
