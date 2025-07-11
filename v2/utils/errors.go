package utils

type BundledPluginError struct {
	PluginName string
	ConfigFile string
}

// New returns an error that formats as the given text.
func NewBundledPluginError(pluginName string, configFile string) error {
	return &BundledPluginError{pluginName, configFile}
}
func (e *BundledPluginError) Error() string {
	return "the plugin '" + e.PluginName + "' is bundled with Discourse"
}
