package resolver

type PluginsResolver struct {
	plugins string
}

// Commit returns the the status of the commit plugin.
func (r PluginsResolver) Commit() bool {
	return len(r.plugins) == 1
}

// Execute returns the the status of the execute plugin.
func (r PluginsResolver) Execute() bool {
	return len(r.plugins) == 1
}
