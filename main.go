package main

import (
	//"fmt"
    //"strconv"
    "github.com/davecgh/go-spew/spew"
)


func main() {

    // Parse
    text := `
    slack {
        channel = #test_
        token = xoxb-77537041330-szNcfH1bvC05LJw8DUOR3s90
        icon_url = https://s3-us-west-2.amazonaws.com/slack-files2/avatars/2015-04-13/4412399782_938284a426b058dc8dd7_36.jpg
    }
    consul {
        alert = slack
        service "consul"
        service "grafana" {
            good = 5
            warn = 5
            crit = 10
        }
    }`
    Parse(text)

    spew.Dump(GetFuse())
    GetFuse().StartChecking(
        GetNotifer(),
    )
}
