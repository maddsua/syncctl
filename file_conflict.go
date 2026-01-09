package syncctl

type ResolvePolicy string

const (
	ResolveSkip      = ResolvePolicy("skip")
	ResolveOverwrite = ResolvePolicy("overwrite")
	ResolveAsCopy    = ResolvePolicy("copy")
)
