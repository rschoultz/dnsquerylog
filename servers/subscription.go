package servers

import (
	"encoding/json"
	"log"
)

type Message struct {
	Type        string
	Url         string
	Exfiltrated string
	Time        string
}

var things = make(map[string]chan string)

func RegisterSubscriber(key string) chan string {
	things[key] = make(chan string, 10)

	return things[key]
}

func RemoveSubscriber(key string) {
	log.Printf("Removing subscriber %s\n", key)
	delete(things, key)
}

func MessageToSubscriber(key string, message Message) bool {

	msg, _ := json.Marshal(message)

	select {
	case things[key] <- string(msg):
		// log.Printf("Message sent for key:%s", key)
		return true
	default:
		// log.Printf("No message sent for key: %s", key)
		return false
	}
}
