package parser

import "sync"

type Result struct {
	// Module is ip command module. for example: ip address show, `address` is the module
	Module string
	// Action is ip command action. for example: ip address show. `show` is the action
	Action string
	// Flags are ip command flags. for example: ip address show dev enp0s3. `dev enp0s3` is the flag
	Properties []Property
}

type Validate func(res *Result)

var ValidateFuncs map[string][]Validate
var once sync.Once

// RegisterValidateFunc ...
func RegisterValidateFunc(module string, fns ...Validate) {
	once.Do(func() {
		ValidateFuncs = make(map[string][]Validate)
	})

	for i := range fns {
		ValidateFuncs[module] = append(ValidateFuncs[module], fns[i])
	}
}

type Parser struct {
}

func (p *Parser) Parse(args []string) error {

}
