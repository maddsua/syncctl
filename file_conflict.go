package syncctl

type ResolvePolicy string

const (
	ResolveSkip       = ResolvePolicy("skip")
	ResolveOverwrite  = ResolvePolicy("overwrite")
	ResolveAsVersions = ResolvePolicy("versions")
)
