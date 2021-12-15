package main

import (
	_ "embed"
	"github.com/urfave/cli/v2"
	"html/template"
	"log"
	"os"
	"sync"

	"dnsquerylog/servers"
)

//go:embed content/index.html
var indexhtml string

var homeTemplate = template.Must(template.New("").Parse(indexhtml))

var isLocal = false

func main() {

	app := &cli.App{}
	app.Copyright = "Copyright 2021, Worldline"
	app.Name = "dnsquerylog"
	app.Usage = ""
	app.HideVersion = true
	app.EnableBashCompletion = true
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "local",
			Value:       false,
			Usage:       "Local server for testing.",
			Destination: &isLocal,
			Required:    false,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	wg := new(sync.WaitGroup)
	wg.Add(3)

	go func() {
		err := servers.ServeUdpNs("53", err)
		if err != nil {
			log.Fatal(err)
			return
		}
	}()

	go func() {
		servers.ServeTcpNs("53", err)
	}()

	go func() {
		servers.ServeWebServers(isLocal, homeTemplate)
	}()

	wg.Wait()

}
