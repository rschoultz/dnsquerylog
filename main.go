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

	log.Printf("Is running locally : %v\n", isLocal)
	log.Printf("TLS                : %v\n", isLocal)
	log.Printf("Serving domain     : %s\n", domain)

	_ = listen.StartServers(isLocal, domain)

}
