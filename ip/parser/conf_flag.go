package parser

type ConfFlag string

const (
	Home          ConfFlag = "home"
	MngtmpAddr    ConfFlag = "mngtmpaddr"
	Nodad         ConfFlag = "nodad"
	Optimistic    ConfFlag = "optimistic"
	NoPrefixRoute ConfFlag = "noprefixroute"
	AutoJoin      ConfFlag = "autojoin"
)
