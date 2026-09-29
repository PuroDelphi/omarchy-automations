package core

import (
	"errors"
	"strings"
)

var errOAuthScopes = errors.New("OAuth authorization does not include every requested scope; reconnect account")

func oauthScopeSet(raw string) (map[string]bool, error) {
	if len(raw) > 8192 {
		return nil, errOAuthResponse
	}
	set := map[string]bool{}
	if raw == "" {
		return set, nil
	}
	for _, scope := range strings.Split(raw, " ") {
		if len(scope) == 0 || len(scope) > 512 {
			return nil, errOAuthResponse
		}
		for _, r := range scope {
			if r < 0x21 || r > 0x7e || r == '"' || r == '\\' {
				return nil, errOAuthResponse
			}
		}
		if set[scope] {
			return nil, errOAuthResponse
		}
		set[scope] = true
	}
	if len(set) > 64 {
		return nil, errOAuthResponse
	}
	return set, nil
}

func oauthScopesInclude(granted string, required []string) bool {
	set, err := oauthScopeSet(granted)
	if err != nil {
		return false
	}
	for _, scope := range required {
		if !set[scope] {
			return false
		}
	}
	return true
}
