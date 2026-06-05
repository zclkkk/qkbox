//go:build !darwin

package capability

func NewNetworkExtensionRuntime(string) NetworkExtensionRuntime {
	return unavailableNetworkExtensionRuntime("Apple NetworkExtension runtime is only available on macOS.")
}
