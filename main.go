package main

import (
	listen "dnsquerylog/cmd"
	_ "embed"
	"flag"
	"log"
)

func main() {
	var isLocal bool
	var domain string
	defaultDomain := "dnsquery.tech"

	flag.BoolVar(&isLocal, "local", false, "Local will not open a https port, only http.")
	flag.StringVar(&domain, "domain", defaultDomain, "Domain name operated by this service. Default: "+defaultDomain)
	flag.Parse()
	log.Printf("Is running locally, without TLS : %v\n", isLocal)
	_ = listen.StartServers(isLocal, domain)

}
