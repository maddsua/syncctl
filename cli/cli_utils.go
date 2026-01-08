package cli

import (
	"fmt"
	"slices"
	"strings"
)

//	todo: mv

type ConflictResolutionPolicy string

const (
	ResolveSkip       = ConflictResolutionPolicy("skip")
	ResolveOverwrite  = ConflictResolutionPolicy("overwrite")
	ResolveAsVersions = ConflictResolutionPolicy("versions")
)

var ConflictFlagValue = &EnumValue{
	Options: []string{string(ResolveSkip), string(ResolveOverwrite), string(ResolveAsVersions)},
	Value:   string(ResolveSkip),
}

type EnumValue struct {
	Options []string
	Value   string
}

func (e *EnumValue) Get() any {
	return e.Value
}

func (e *EnumValue) Set(value string) error {
	if slices.Contains(e.Options, value) {
		e.Value = value
		return nil
	}
	return fmt.Errorf("allowed values are %s", strings.Join(e.Options, ", "))
}

func (e *EnumValue) String() string {
	return e.Value
}
