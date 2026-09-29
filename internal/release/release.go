// Package release describes independently checked component contracts.
package release

import "quatrro.local/automations/internal/local"

const Version = "0.1.0-dev"
const UIContract = 1

type Descriptor struct {
	Component  string `json:"component"`
	Version    string `json:"version"`
	Protocol   int    `json:"protocol"`
	Contract   int    `json:"contract"`
	UIContract int    `json:"ui_contract"`
}

func Describe(component string) Descriptor {
	return Descriptor{Component: component, Version: Version, Protocol: local.Version, Contract: local.Contract, UIContract: UIContract}
}
