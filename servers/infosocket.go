package servers

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/acme/autocert"
	"html/template"
	"log"
	"net/http"
	"time"
)

const wsPath = "/wsConnection"

var wsProtocol = "wss"
var addr = flag.String("addr", "0.0.0.0:80", "http service address")
var indexhtml *template.Template
var upgrader = websocket.Upgrader{} // use default options

func wsConnection(w http.ResponseWriter, r *http.Request) {

	id := uuid.New()
	idString := id.String()
	checkAddr := fmt.Sprintf("%s.check.dnsquery.tech", idString)
	subscriber := idString

	log.Printf("Upgrading to Websocket listener for id: %s\n", id)

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer func() {
		RemoveSubscriber(subscriber)
		c.Close()
	}()

	go func(ws *websocket.Conn) {
		newAddress := Message{
			Type: "newaddress",
			Url:  checkAddr,
			Time: time.Now().Format(time.RFC3339),
		}
		jsonMessage, _ := json.Marshal(newAddress)
		ws.WriteMessage(websocket.TextMessage, []byte(jsonMessage))
		for {
			if _, _, err := ws.NextReader(); err != nil {
				RemoveSubscriber(subscriber)
				ws.Close()
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

func ServeWebServers(isLocal bool, homeTemplate *template.Template) {

	indexhtml = homeTemplate

	if isLocal {
		wsProtocol = "ws"
		httpHandlers()
		go http.ListenAndServe("0.0.0.0:http", nil)
		return
	}

	go func() {
		flag.Parse()
		log.SetFlags(0)

		certManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist("dnsquery.tech", "check.dnsquery.tech", "n2.dnsquery.tech", "view.dnsquery.tech"),
			Cache:      autocert.DirCache("certs"),
		}

		httpsServer := &http.Server{
			Addr: "0.0.0.0:https",
			TLSConfig: &tls.Config{
				GetCertificate: certManager.GetCertificate,
			},
		}

		httpHandlers()

		go http.ListenAndServe("0.0.0.0:http", certManager.HTTPHandler(nil))
		log.Fatal(httpsServer.ListenAndServeTLS("", ""))
	}()
}

func httpHandlers() {
	http.HandleFunc(wsPath, wsConnection)
	http.HandleFunc("/", home)
}

func home(w http.ResponseWriter, r *http.Request) {

	err := indexhtml.Execute(w, wsProtocol+"://"+r.Host+wsPath)
	if err != nil {
		log.Fatal(err)
		return
	}
}
