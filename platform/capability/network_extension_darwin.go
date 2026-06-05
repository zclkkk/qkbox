//go:build darwin

package capability

func NewNetworkExtensionRuntime(string) NetworkExtensionRuntime {
	return unavailableNetworkExtensionRuntime("Apple NetworkExtension container is not installed or authorized.")
}
