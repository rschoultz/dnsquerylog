package servers

import (
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/acme/autocert"
	"html/template"
	"log"
	"net"
	"net/http"
	"time"
)

const wsPath = "/wsConnection"

//go:embed content/index.html
var indexHtml string
var homeTemplate = template.Must(template.New("").Parse(indexHtml))

var wsProtocol = "wss"
var upgrader = websocket.Upgrader{} // use default options

func wsConnection(w http.ResponseWriter, r *http.Request, domain string) {

	id := uuid.New()
	idString := id.String()
	checkAddr := fmt.Sprintf("%s.check."+domain, idString)
	subscriber := idString
	requesterIp := r.RemoteAddr

	log.Printf("Upgrading to Websocket listener for ip:%s id: %s\n", requesterIp, id)

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer func() {
		log.Printf("Closing Websocket ip:%s id: %s\n", requesterIp, id)
		RemoveSubscriber(subscriber)
		_ = c.Close()
	}()

	go func(ws *websocket.Conn) {
		newAddress := Message{
			Type: "newaddress",
			Url:  checkAddr,
			Time: time.Now().Format(time.RFC3339),
		}
		jsonMessage, _ := json.Marshal(newAddress)
		_ = ws.WriteMessage(websocket.TextMessage, []byte(jsonMessage))
		for {
			if _, _, err := ws.NextReader(); err != nil {
				RemoveSubscriber(subscriber)
				_ = ws.Close()
				break
			}
		}
	}(c)

	channelConsumer := RegisterSubscriber(subscriber)

	for {
		var message string = <-channelConsumer
		err = c.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func ServeWebServers(isLocal bool, domain string) (err error) {

	if isLocal {
		wsProtocol = "ws"
		httpHandlers(domain)
		log.Printf("Starting http server.")
		go func() error {
			err = http.ListenAndServe(":http", nil)
			return err
		}()
		return nil
	}

	go func() {
		flag.Parse()
		log.SetFlags(0)

		certManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(domain, "check."+domain, "*.check."+domain, "n2."+domain, "view"+domain),
			Cache:      autocert.DirCache("certs"),
		}

		log.Printf("Starting https server.")
		httpsServer := &http.Server{
			Addr: ":https",
			TLSConfig: &tls.Config{
				GetCertificate: certManager.GetCertificate,
			},
		}

		httpHandlers(domain)

		log.Printf("Starting http server.")
		go func() error {
			err := http.ListenAndServe(":http", certManager.HTTPHandler(nil))
			if err != nil {
				log.Fatal(err)
				return err
			}
			return nil
		}()
		log.Fatal(httpsServer.ListenAndServeTLS("", ""))
	}()
	return nil
}

func httpHandlers(domain string) {
	http.HandleFunc("/", home)
	http.HandleFunc(wsPath, myHttpHandler(domain))

}

func myHttpHandler(domain string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wsConnection(w, r, domain)
	}
}

func home(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()

	srvAddr := ctx.Value(http.LocalAddrContextKey).(net.Addr)
	fmt.Println(r.Host)
	fmt.Println(r.Proto)
	fmt.Println(srvAddr)

	err := homeTemplate.Execute(w, wsProtocol+"://"+r.Host+wsPath)
	if err != nil {
		log.Fatal(err)
		return
	}
}
