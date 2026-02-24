package tools

import (
	"net"
	"net/url"
)

// IsInternalIP checks if the given IP address is an internal IP address (e.g., loopback, private, link-local, or multicast).
// IsLoopback — 127.0.0.0/8, ::1
// IsPrivate — 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, fc00::/7
// IsLinkLocalUnicast — 169.254.0.0/16, fe80::/10 (covers cloud metadata 169.254.169.254)
// IsLinkLocalMulticast — 224.0.0.0/24, ff02::/16
// IsUnspecified — 0.0.0.0, ::
// IsMulticast — 224.0.0.0/4, ff00::/8
func IsInternalIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() ||
		ip.IsMulticast()
}

// IsValidLinkURL checks if the provided string is a valid URL with http or https scheme and a non-empty hostname.
func IsValidLinkURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != ""
}
