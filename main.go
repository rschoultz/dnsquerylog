package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"sync"

	"github.com/miekg/dns"
	"github.com/urfave/cli/v2"

	"dnsquerylog/servers"
)

const defaultARecord = "*.check.dnsquery.tech."

var aRecords = map[string]string{
	"n2.dnsquery.tech.":      "35.217.6.93",
	"*.check.dnsquery.tech.": "35.217.6.93",
}

var nsRecords = map[string]string{
	"check.dnsquery.tech.": "n2.dnsquery.tech.",
}

var soaRecords = map[string]string{
	"check.dnsquery.tech.": "n2.dnsquery.tech. hostmaster.dnsquery.tech. 1 21600 3600 259200 300",
}

func parseQuery(m *dns.Msg, requester net.Addr) {
	for _, q := range m.Question {
		log.Printf("Query: %s of type %d (via %s) - details: %s\n", q.Name, q.Qtype, requester.String(), q.String())
		////clients.SendMessage(q.Name)

		exfiltrated, subscriber, err := extractUrlParts(q.Name)
		if err {
			return
		}

		log.Printf("Substrings of query 1: %s and 2: %s", exfiltrated, subscriber)

		subscriberListened := servers.MessageToSubscriber(subscriber, q.Name)

		if !subscriberListened {
			log.Printf("Should respond with nothing here, noone is watching results.")
			return
		}

		switch q.Qtype {
		case dns.TypeA:
			ip := aRecords[q.Name]
			if ip == "" {
				ip = aRecords[defaultARecord]
			}
			rr, err := dns.NewRR(fmt.Sprintf("%s A %s", q.Name, ip))
			if err == nil {
				m.Answer = append(m.Answer, rr)
			}
		case dns.TypeNS:
			ns := nsRecords[q.Name]
			if ns != "" {
				rr, err := dns.NewRR(fmt.Sprintf("%s NS %s", q.Name, ns))
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}
		case dns.TypeSOA:
			soa := soaRecords[q.Name]
			if soa != "" {
				rr, err := dns.NewRR(fmt.Sprintf("%s SOA %s", q.Name, soa))
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}
		}
	}
}

func extractUrlParts(q string) (string, string, bool) {
	regex := *regexp.MustCompile("(.*)(.{36}).check.dnsquery.tech")
	res := regex.FindStringSubmatch(q)
	if res == nil {
		log.Printf("Illegal query address: %s", q)
		return "", "", true
	}

	exfiltrated := res[1]
	subscriber := res[2]
	return exfiltrated, subscriber, false
}

func handleDnsRequest(w dns.ResponseWriter, r *dns.Msg) {

	defer w.Close()

	requesterAddr := w.RemoteAddr()
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		parseQuery(m, requesterAddr)
	}

	w.WriteMsg(m)

}

func main() {

	var nsPort string

	app := &cli.App{}
	app.Copyright = "Copyright 2021, Worldline"
	app.Name = "dnsquerylog"
	app.Usage = ""
	app.HideVersion = true
	app.EnableBashCompletion = true
	key := ""
	addr := ""
	cert := ""
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "port",
			Aliases:     []string{"p"},
			Value:       "53",
			Usage:       "port number to listen on.",
			Destination: &nsPort,
			Required:    false,
		},
		&cli.StringFlag{
			Name:        "addr",
			Value:       ":443",
			Usage:       "HTTP/2 Address to listen on.",
			Destination: &addr,
			Required:    false,
		},
		&cli.StringFlag{
			Name:        "cert",
			Value:       "53",
			Usage:       "Certificate for HTTP/2.",
			Destination: &cert,
			Required:    false,
		},
		&cli.StringFlag{
			Name:        "key",
			Value:       "53",
			Usage:       "Key for HTTP/2.",
			Destination: &key,
			Required:    false,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

	// start server

	//	log.Printf("Starting HTTP2 server.\n")
	//	myh2.My_http2(":443", "cert.pem", "key.pem")

	wg := new(sync.WaitGroup)
	wg.Add(3)

	dns.HandleFunc("dnsquery.tech.", handleDnsRequest)

	go func() {
		err = ServeUdpNs(nsPort, err)
	}()

	go func() {
		serveTcpNs(nsPort, err)
	}()

	go func() {
		servers.ServeWebServers()
	}()

	wg.Wait()

}

func serveTcpNs(nsPort string, err error) {
	log.Printf("Starting TCP ns at %s\n", nsPort)
	serverTcp := &dns.Server{Addr: ":" + nsPort, Net: "tcp"}
	err = serverTcp.ListenAndServe()
	defer serverTcp.Shutdown()
	if err != nil {
		log.Fatalf("Failed to start TCP server: %s\n ", err.Error())
	}
}

func ServeUdpNs(nsPort string, err error) error {
	log.Printf("Starting UDP ns at %s\n", nsPort)
	serverUdp := &dns.Server{Addr: ":" + nsPort, Net: "udp"}
	err = serverUdp.ListenAndServe()
	defer serverUdp.Shutdown()
	if err != nil {
		log.Fatalf("Failed to start UDP server: %s\n ", err.Error())
	}
	return err
}
