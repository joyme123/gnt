package cmd

import (
	"flag"
	"os"

	"github.com/go-logr/glogr"
	"github.com/go-logr/logr"
)

var DebugLogger logr.Logger

func init() {
	flag.Parse()
	os.Stderr = os.Stdout
	DebugLogger = glogr.New()
}
