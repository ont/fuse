package main

import (
	"fmt"
	"os"
	//"log"
	"io/ioutil"
	//"strconv"
	//"github.com/davecgh/go-spew/spew"
	"fuse/pkg/domain"
	"fuse/pkg/monitor"
	"fuse/pkg/parser"

	log "github.com/sirupsen/logrus"
)

func main() {
	// check args
	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stderr, "usage: fuse [config]")
		fmt.Fprintln(os.Stderr, "error: no config specified")
		os.Exit(1)
	}
	if os.Args[1] == "-v" {
		log.SetLevel(log.DebugLevel)
	}

	// load config
	bytes, err := ioutil.ReadFile(os.Args[len(os.Args)-1])

	// parse config
	result, err := parser.Parse(string(bytes))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error during parsing config file:", err)
		os.Exit(1)
	}

	// prepare notifier
	notifer := domain.NewNotifer()
	for name, alerter := range result.Alerters {
		notifer.AddAlerter(name, alerter)
	}

	// prepare monitors and create fuse
	fuse := monitor.NewFuse()
	for _, monitor := range result.Monitors {
		fuse.AddMonitor(monitor)
	}

	// start monitor's gorutines and wait
	fuse.RunWith(notifer)
}
