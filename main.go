package main

import (
    "os"
    "fmt"
    "io/ioutil"
    //"strconv"
    //"github.com/davecgh/go-spew/spew"
)


func main() {
    // load config
    if len(os.Args) == 1 {
        fmt.Fprintln(os.Stderr, "error: no config specified")
        fmt.Fprintln(os.Stderr, "usage: fuse [config]")
        os.Exit(1)
    }
    bytes, err := ioutil.ReadFile(os.Args[1])

    // parse config
    result, err := Parse(string(bytes))
    if err != nil {
        fmt.Fprintln(os.Stderr, "error during parsing config file:", err)
        os.Exit(1)
    }

    // prepare notifier
    notifer := NewNotifer()
    for name, alerter := range result.Alerters {
        notifer.AddAlerter(name, alerter)
    }

    // prepare monitors and create fuse
    fuse := NewFuse()
    for _, monitor := range result.Monitors {
        fuse.AddMonitor(monitor)
    }

    // start monitor's gorutines and wait
    fuse.RunWith(notifer)
}
