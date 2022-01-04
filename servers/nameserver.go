package servers

import (
	"fmt"
	"github.com/miekg/dns"
	"log"
	"net"
	"regexp"
	"strings"
	"time"
)

// FIXME: Only halfway through in fixing hard coded stuff
const defaultARecord = "*.check."
const ttl = "30"

var DEBUG = false

var aRecords = map[string]string{
	"n2.":      "35.217.6.93",
	"*.check.": "35.217.6.93",
}

var nsRecords = map[string]string{
	"check.": "n2.",
}

var soaRecords = map[string]string{
	"check.": "300 IN SOA n2.DOMAIN. hostmaster.DOMAIN. 1 21600 3600 259200 300",
}

func parseQuery(m *dns.Msg, requester net.Addr, domain string) {
	currentTime := time.Now().Format(time.RFC3339)

	for _, q := range m.Question {
		requesterIp := requester.String()
		exfiltrated, subscriber, _ := extractUrlParts(q.Name, "check."+domain)

		lookupString := strings.Replace(q.Name, domain+".", "", 1)

		if DEBUG {
			log.Printf("lookupstring:%s, name:%s, qtype:%s, qclass:%s, subscriber:%s\n", lookupString, q.Name, q.Qtype, q.Qclass, subscriber)
		}

		var subscriberListened bool
		if subscriber != "" {

			queryType := dns.TypeToString[q.Qtype]

			depletedUrl := false
			if exfiltrated == "" {
				depletedUrl = true
			}

			message := Message{
				Type:        "lookup",
				Url:         q.Name,
				DepletedUrl: depletedUrl,
				Exfiltrated: exfiltrated,
				Protocol:    "dns",
				Query:       queryType,
				Time:        currentTime,
			}

			subscriberListened = MessageToSubscriber(subscriber, message)
			if !subscriberListened {
				return
			}
		}

		//TODO: Remove this - we don't want sensitive data being logged.
		log.Printf("%s LOOKUP by:%s wss:%t using:%s (%s) for:%s got additional: %s\n",
			currentTime, requesterIp, !subscriberListened, subscriber, dns.TypeToString[q.Qtype], lookupString, exfiltrated)

		switch q.Qtype {
		case dns.TypeA:
			ip := aRecords[lookupString]
			if ip == "" {
				ip = aRecords[defaultARecord]
			}
			aRecord := fmt.Sprintf("%s %s IN A %s", q.Name, ttl, ip)
			rr, err := dns.NewRR(aRecord)
			if err == nil {
				m.Answer = append(m.Answer, rr)
				m.Authoritative = true
			}
		case dns.TypeNS:
			ns := nsRecords[lookupString]
			if ns != "" {
				rr, err := dns.NewRR(fmt.Sprintf("%s NS %s", q.Name, ns+domain+"."))
				if err == nil {
					m.Answer = append(m.Answer, rr)
					m.Authoritative = true
				}
			}
		case dns.TypeSOA:
			inspect("SOA lookup for", lookupString)
			soa := soaRecords[lookupString]
			if soa != "" {
				soaRR := fmt.Sprintf("%s %s", q.Name, soa)
				domainSoaRR := strings.Replace(soaRR, "DOMAIN", domain, -1)

				inspect("Produced SOA RR", domainSoaRR)
				rr, err := dns.NewRR(domainSoaRR)
				if err == nil {
					m.Answer = append(m.Answer, rr)
					m.Authoritative = true
				}
			}
		}
	}
}

func extractUrlParts(q string, checkDomain string) (string, string, error) {
	regex := *regexp.MustCompile("(.*)(.{36})." + checkDomain)
	res := regex.FindStringSubmatch(q)
	if res == nil {
		return "", "", fmt.Errorf("not a valid query address: %s", q)
	}

	exfiltrated := res[1]
	subscriber := res[2]
	return exfiltrated, subscriber, nil
}

func dnsHandlerWithDomain(domain string) dns.HandlerFunc {
	return func(w dns.ResponseWriter, r *dns.Msg) {
		handleDnsRequest(w, r, domain)
	}
}

func handleDnsRequest(w dns.ResponseWriter, r *dns.Msg, domain string) {

	defer func(w dns.ResponseWriter) {
		_ = w.Close()
	}(w)

	requesterAddr := w.RemoteAddr()
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	switch r.Opcode {
	case dns.OpcodeQuery:
		inspect("DNS msg ", r)
		parseQuery(m, requesterAddr, domain)
	}

	err := w.WriteMsg(m)
	if err != nil {
		log.Println(err)
		return
	}

}

func ServeTcpNs(nsPort string, domain string) error {
	dns.HandleFunc(domain+".", dnsHandlerWithDomain(domain))

	log.Printf("Starting TCP ns at %s\n", nsPort)
	serverTcp := &dns.Server{Addr: ":" + nsPort, Net: "tcp"}
	err := serverTcp.ListenAndServe()
	defer func(serverTcp *dns.Server) {
		_ = serverTcp.Shutdown()
	}(serverTcp)
	if err != nil {
		log.Printf("Failed to start TCP server: %s\n ", err.Error())
	}
	return err
}

func ServeUdpNs(nsPort string, domain string) error {
	dns.HandleFunc(domain+".", dnsHandlerWithDomain(domain))

	log.Printf("Starting UDP ns at %s\n", nsPort)
	serverUdp := &dns.Server{Addr: ":" + nsPort, Net: "udp"}
	err := serverUdp.ListenAndServe()
	defer func(serverUdp *dns.Server) {
		_ = serverUdp.Shutdown()
	}(serverUdp)
	if err != nil {
		log.Fatalf("Failed to start UDP server: %s\n ", err.Error())
	}
	return err
}

func inspect(v ...interface{}) {
	if DEBUG {
		fmt.Println(">>> Inspecting:")
		for _, v := range v {
			fmt.Printf("%T %#v \n", v, v)
		}
		fmt.Println("<<<")
	}
}
