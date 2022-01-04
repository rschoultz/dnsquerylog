package servers

import (
	"encoding/json"
	"log"
)

type Message struct {
	Type        string
	Url         string
	DepletedUrl bool
	Exfiltrated string
	Query       string
	Time        string
	Protocol    string
}

var subscribers = make(map[string]chan string)

func RegisterSubscriber(key string) chan string {
	subscribers[key] = make(chan string, 10)

	return subscribers[key]
}

func RemoveSubscriber(key string) {
	log.Printf("Removing subscriber %s\n", key)
	delete(subscribers, key)
}

func MessageToSubscriber(key string, message Message) bool {
	msg, _ := json.Marshal(message)

	select {
	case subscribers[key] <- string(msg):
		// log.Printf("Message sent for key:%s", key)
		return true
	default:
		// log.Printf("No message sent for key: %s", key)
		return false
	}
}
