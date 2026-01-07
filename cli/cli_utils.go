package cli

type FileConflicResolution int

const (
	ResolveSkip = FileConflicResolution(iota)
	ResolveOverwrite
	ResolveAsVersions
)
