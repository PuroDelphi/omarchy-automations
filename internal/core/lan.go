package core

import (
	"errors"
	"net"
	"net/netip"
	"net/url"
	"strconv"
)

var errPrivateEndpoint = errors.New("private network exception must match the destination's exact host and port")

func destinationAuthority(d Destination) string {
	u, err := url.Parse(d.URL)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
		return ""
	}
	port := u.Port()
	if port == "" {
		port = "443"
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 || strconv.Itoa(number) != port {
		return ""
	}
	return net.JoinHostPort(u.Hostname(), port)
}

func validatePrivateEndpoints(d Destination) error {
	if len(d.PrivateHosts) == 0 {
		return nil
	}
	if len(d.PrivateHosts) != 1 || d.PrivateHosts[0] != destinationAuthority(d) || d.PrivateHosts[0] == "" {
		return errPrivateEndpoint
	}
	return nil
}

// An endpoint grant allows local unicast only. Multicast, unspecified,
// documentation and transition ranges remain blocked even with a grant.
func privateUnicast(ip netip.Addr) bool {
	ip = ip.Unmap()
	return ip.IsValid() && (ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || netip.MustParsePrefix("100.64.0.0/10").Contains(ip))
}
