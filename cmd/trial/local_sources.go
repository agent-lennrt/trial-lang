package main

import (
	"context"
	"fmt"

	"github.com/lennrt/trial-lang/canon"
	"github.com/lennrt/trial-lang/internal/docket"
)

// loadLocalSources shares source bounds and dependency order between run and
// check. Callers validate arguments and allow stdin for at most one source.
func loadLocalSources(ctx context.Context, path string, enactments []string, useCanon bool) (string, []string, error) {
	statutes := make([]string, 0, len(enactments)+len(canon.Files()))
	if useCanon {
		for _, name := range canon.Files() {
			source, err := canon.FS.ReadFile(name)
			if err != nil {
				return "", nil, fmt.Errorf("read canon: %w", err)
			}
			statutes = append(statutes, string(source))
		}
	}
	totalStatuteBytes := 0
	for _, name := range enactments {
		if err := ctx.Err(); err != nil {
			return "", nil, err
		}
		source, err := readSource(ctx, name)
		if err != nil {
			return "", nil, fmt.Errorf("read statute: %w", err)
		}
		totalStatuteBytes += len(source)
		if totalStatuteBytes > docket.MaxReadBytes {
			return "", nil, fmt.Errorf("statutes exceed %d bytes", docket.MaxReadBytes)
		}
		statutes = append(statutes, string(source))
	}
	if err := ctx.Err(); err != nil {
		return "", nil, err
	}
	source, err := readSource(ctx, path)
	if err != nil {
		return "", nil, fmt.Errorf("read program: %w", err)
	}
	return string(source), statutes, nil
}
