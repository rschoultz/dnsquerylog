package main

import (
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"net"
	"os"
	"regexp"
	"sync"
	"time"

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

//go:embed content/index.html
var indexhtml string

var homeTemplate = template.Must(template.New("").Parse(indexhtml))

func parseQuery(m *dns.Msg, requester net.Addr) {
	for _, q := range m.Question {
		requesterIp := requester.String()
		exfiltrated, subscriber, err := extractUrlParts(q.Name)
		if err {
			return
		}
		currentTime := time.Now().Format(time.RFC3339)
		message := servers.Message{
			Type:        "lookup",
			Url:         q.Name,
			Exfiltrated: exfiltrated,
			Time:        currentTime,
		}

		subscriberListened := servers.MessageToSubscriber(subscriber, message)

		log.Printf("%s LOOKUP for:%s attempt-stealth:%t using:%s (qtype:%d) additional: %s\n",
			currentTime, requesterIp, !subscriberListened, subscriber, q.Qtype, exfiltrated)

		if !subscriberListened {
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

	dns.HandleFunc("dnsquery.tech.", handleDnsRequest)

	go func() {
		err = ServeUdpNs("53", err)
	}()

	go func() {
		serveTcpNs("53", err)
	}()

	go func() {
		servers.ServeWebServers(isLocal, homeTemplate)
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
