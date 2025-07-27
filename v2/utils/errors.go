package utils

type BundledPluginError struct {
	ParentError error
	PluginName  string
	ConfigFile  string
}

// New returns an error that formats as the given text.
func NewBundledPluginError(parentError error, pluginName string, configFile string) error {
	return &BundledPluginError{parentError, pluginName, configFile}
}
func (e *BundledPluginError) Error() string {
	return e.ParentError.Error() + ": the plugin '" + e.PluginName + "' is bundled with Discourse"
}
