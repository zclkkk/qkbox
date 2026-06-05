# macOS Packaging

The macOS baseline is the Wails app bundle plus the formal Apple
NetworkExtension runtime container path for VPN/TUN mode.

Release packaging must follow Apple signing, notarization, and entitlement
requirements. qkbox does not ship macOS TUN mode through a root route mutation
installer or ad-hoc privileged helper.
