package parser

type ScopeID string

const (
	Host   ScopeID = "host"
	Link   ScopeID = "link"
	Global ScopeID = "global"
)
