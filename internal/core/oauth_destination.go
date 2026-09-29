package core

import (
	"errors"
	"net/url"
)

var errOAuthDestination = errors.New("Google OAuth requires an approved Google API HTTPS destination without LAN exceptions")

// Explicit initial API hosts, not a suffix match: user-controlled storage hosts
// and lookalike domains must never receive the account's bearer token.
func validateGoogleDestination(d Destination) error {
	u, err := url.Parse(d.URL)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Fragment != "" || (u.Port() != "" && u.Port() != "443") || len(d.PrivateHosts) != 0 {
		return errOAuthDestination
	}
	switch u.Hostname() {
	case "www.googleapis.com", "calendar.googleapis.com":
	default:
		return errOAuthDestination
	}
	return nil
}
