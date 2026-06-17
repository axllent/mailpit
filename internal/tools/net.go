package tools

import (
	"encoding/binary"
	"net"
	"net/url"
)

var (
	// cgnatRange is the CGNAT shared address space (RFC 6598), not covered by net.IP.IsPrivate().
	// CGNAT (Carrier-Grade NAT) is a technique used by ISPs to conserve IPv4 addresses. Instead of assigning a unique
	// public IP to every customer, the ISP places many customers behind a shared NAT, then gives them all addresses
	// from the reserved 100.64.0.0/10 range (RFC 6598) on their internal network.
	cgnatRange = mustCIDR("100.64.0.0/10")

	// IPv6 transition prefixes that embed an IPv4 destination. Go's net.IP.Is* family
	// does not decode these, so an IPv6 literal of one of these forms can carry a
	// private/link-local IPv4 destination past the stdlib checks. See golang/go#79925.
	nat64WellKnown = mustCIDR("64:ff9b::/96")   // RFC 6052
	nat64LocalUse  = mustCIDR("64:ff9b:1::/48") // RFC 8215
	sixToFour      = mustCIDR("2002::/16")      // RFC 3056
	teredo         = mustCIDR("2001::/32")      // RFC 4380
	ipv4Compatible = mustCIDR("::/96")          // RFC 4291 §2.5.5.1
	// IPv4-mapped IPv6 (::ffff:0:0/96, RFC 4291 §2.5.5.2) is normalised by net.IP.To4,
	// so the stdlib Is* checks above already see the embedded IPv4 - no decode needed.

	// Direct IPv6 prefixes outside the scope of Go's stdlib Is* family.
	deprecatedSiteLocal = mustCIDR("fec0::/10")     // RFC 3879 / RFC 4291 §2.5.7 — deprecated, still routable on dual-stack hosts
	documentationPrefix = mustCIDR("2001:db8::/32") // RFC 3849 — documentation only, must not appear in real traffic
)

// MustCIDR is a helper for use in global var initialisation.
func mustCIDR(s string) *net.IPNet {
	_, cidr, _ := net.ParseCIDR(s)

	return cidr
}

// IsInternalIP checks if the given IP address is an internal IP address (e.g., loopback, private, link-local, or multicast).
// IsLoopback - 127.0.0.0/8, ::1
// IsPrivate - 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, fc00::/7
// IsLinkLocalUnicast - 169.254.0.0/16, fe80::/10 (covers cloud metadata 169.254.169.254)
// IsLinkLocalMulticast - 224.0.0.0/24, ff02::/16
// IsUnspecified - 0.0.0.0, ::
// IsMulticast - 224.0.0.0/4, ff00::/8
// CGNAT - 100.64.0.0/10 (RFC 6598) (Carrier-Grade NAT)
// IPv6 transition forms - NAT64 (RFC 6052/8215), 6to4 (RFC 3056), Teredo (RFC 4380),
// IPv4-compatible (RFC 4291) - re-checked against their embedded IPv4.
func IsInternalIP(ip net.IP) bool {
	if ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() ||
		ip.IsMulticast() ||
		cgnatRange.Contains(ip) ||
		deprecatedSiteLocal.Contains(ip) ||
		documentationPrefix.Contains(ip) {
		return true
	}

	if embeddedV4, ok := embeddedIPv4(ip); ok {
		return IsInternalIP(embeddedV4)
	}

	return false
}

// embeddedIPv4 returns the IPv4 destination encoded in ip, if ip is an IPv6 form
// documented to carry one. Without this, an IPv6 literal like 64:ff9b::a9fe:a9fe
// (NAT64 wrapping 169.254.169.254) bypasses the stdlib Is* checks above.
func embeddedIPv4(ip net.IP) (net.IP, bool) {
	// Skip addresses that are already IPv4 (4-byte or IPv4-mapped IPv6) - those are
	// covered by the stdlib Is* checks via To4 normalisation. Re-entering here would
	// recurse infinitely, because To16 turns an IPv4 back into ::ffff:<ipv4>.
	if ip.To4() != nil {
		return nil, false
	}

	ip16 := ip.To16()
	if ip16 == nil || len(ip16) != net.IPv6len {
		return nil, false
	}

	switch {
	case nat64WellKnown.Contains(ip16), nat64LocalUse.Contains(ip16),
		ipv4Compatible.Contains(ip16):
		// Last 32 bits are the embedded IPv4.
		return net.IPv4(ip16[12], ip16[13], ip16[14], ip16[15]).To4(), true
	case sixToFour.Contains(ip16):
		// Bits 16..47 are the embedded IPv4.
		return net.IPv4(ip16[2], ip16[3], ip16[4], ip16[5]).To4(), true
	case teredo.Contains(ip16):
		// Bits 96..127 are the embedded IPv4 XOR'd with 0xFFFFFFFF.
		x := binary.BigEndian.Uint32(ip16[12:16]) ^ 0xFFFFFFFF
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, x)
		return net.IPv4(b[0], b[1], b[2], b[3]).To4(), true
	case ip16[10] == 0x5e && ip16[11] == 0xfe:
		// ISATAP (RFC 5214) - interface identifier ends with :5efe:<ipv4>. The /64
		// prefix is not fixed (any subnet can carry ISATAP), so match structurally
		// on bytes 10-11 and treat bytes 12-15 as the embedded IPv4. Must run after
		// the fixed-prefix cases above (Teredo can legitimately have 5efe in bytes
		// 10-11; its embedding takes precedence).
		return net.IPv4(ip16[12], ip16[13], ip16[14], ip16[15]).To4(), true
	}

	return nil, false
}

// IsValidLinkURL checks if the provided string is a valid URL with http or https scheme and a non-empty hostname.
func IsValidLinkURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != ""
}
