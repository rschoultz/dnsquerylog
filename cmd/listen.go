package listen

import (
	"dnsquerylog/servers"
	"log"
	"sync"
)

func StartServers(isLocal bool, domain string) (err error) {
	wg := new(sync.WaitGroup)
	wg.Add(3)

	go func() {
		err := servers.ServeUdpNs("53", domain)
		if err != nil {
			log.Fatal(err)
			return
		}
	}()

	go func() {
		err := servers.ServeTcpNs("53", domain)
		if err != nil {
			log.Fatal(err)
			return
		}
	}()

	go func() {
		err := servers.ServeWebServers(isLocal, domain)
		if err != nil {
			log.Fatal(err)
			return
		}
	}()

	wg.Wait()

	return err
}
