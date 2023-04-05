package parser

type PropertyType string

const (
	Dev         PropertyType = "dev"
	Local       PropertyType = "local"
	Peer        PropertyType = "peer"
	Broadcast   PropertyType = "broadcast"
	Label       PropertyType = "label"
	Scope       PropertyType = "scope"
	Metric      PropertyType = "metric"
	ValidLFT    PropertyType = "valid_lft"
	PreferreLFT PropertyType = "preferred_lft"
	To          PropertyType = "to"
	Master      PropertyType = "master"
	Vrf         PropertyType = "vrf"
	Type        PropertyType = "type"
)

type Property struct {
	Name  string
	Value string
}
