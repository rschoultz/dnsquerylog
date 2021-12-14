package servers

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/acme/autocert"
	"html/template"
	"log"
	"net/http"
)

const wsPath = "/wsConnection"

var addr = flag.String("addr", "0.0.0.0:80", "http service address")

var upgrader = websocket.Upgrader{} // use default options

func wsConnection(w http.ResponseWriter, r *http.Request) {

	id := uuid.New()
	idString := id.String()
	checkAddr := fmt.Sprintf("%s.check.dnsquery.tech.", idString)
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
		ws.WriteMessage(websocket.TextMessage, []byte("Use this address: "+checkAddr))
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

func ServeWebServers() {

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

		http.HandleFunc(wsPath, wsConnection)
		http.HandleFunc("/", home)

		go http.ListenAndServe("0.0.0.0:http", certManager.HTTPHandler(nil))

		log.Fatal(httpsServer.ListenAndServeTLS("", ""))
	}()
}

func home(w http.ResponseWriter, r *http.Request) {
	err := homeTemplate.Execute(w, "wss://"+r.Host+wsPath)
	if err != nil {
		log.Fatal(err)
		return
	}
}

var homeTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<script>  
window.addEventListener("load", function(evt) {
    var output = document.getElementById("output");
    var input = document.getElementById("input");
    var ws;

    var print = function(message) {
        var d = document.createElement("div");
        d.textContent = message;
        output.appendChild(d);
        output.scroll(0, output.scrollHeight);
    };

    var closeWs = function(ws) {
         if (!ws) {
            return false;
        }
        ws.close();
	};

    document.getElementById("open").onclick = function(evt) {
        if (ws) {
            return false;
        }
        ws = new WebSocket("{{.}}");
        ws.onopen = function(evt) {
            print("Creating network endpoint...");
        }
        ws.onclose = function(evt) {
            print("Closed network endpoint.");
            ws = null;
        }
        ws.onmessage = function(evt) {
            print("> " + evt.data);
        }
        ws.onerror = function(evt) {
            print("ERROR: " + evt.data);
        }
        return false;
    };
    document.getElementById("close").onclick = function(evt) {
        if (!ws) {
            return false;
        }
        ws.close();
        return false;
    };
});
</script>
</head>
<body>
<table>
<tr><td valign="top" width="50%">
<p>Click "Get address" get a working address, and "Close" close when you are done. 
<p>
<form>
<button id="open">Get address</button>
<button id="close">Close</button>
</form>
</td><td valign="top" width="50%">
<div id="output" style="max-height: 170vh;overflow-y: scroll;"></div>
</td></tr></table>
</body>
</html>
`))
