package parser

type FlagType string

const (
	Permanent  FlagType = "permanent"
	Dynamic    FlagType = "dynamic"
	Primary    FlagType = "primary"
	Secondary  FlagType = "secondary"
	Tentative  FlagType = "tentative"
	Deprecated FlagType = "deprecated"
	DadFailed  FlagType = "dadfailed"
	Temporary  FlagType = "temporary"
)
