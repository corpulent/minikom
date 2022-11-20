package main

import (
	"log"
	"math/rand"
	"time"
)

var (
	states   = []string{"issue", "deploy"}
	services = []string{"web-server", "database", "cache", "producer", "consumer", "message-broker"}
	endpoint = getenv("ENDPOINT")
)

func sendEvent(serviceName string, state string) {
	timestamp := time.Now().Unix()
	delay(rand.Intn(3) == 0, 10)
	postJSON(endpoint, map[string]interface{}{
		"service_name": serviceName,
		"state":        state,
		"timestamp":    timestamp,
	})
}

func monitorService(serviceName string) {
	for {
		state := states[rand.Intn(len(states))]
		duration := 1 + rand.Intn(60)
		for i := 0; i < duration; i++ {
			go sendEvent(serviceName, state)
			time.Sleep(1 * time.Second)
		}
		delay(true, 120)
	}
}

func main() {
	log.Println("Starting komobox")
	log.Printf("Redirecting events to %s\n", endpoint)

	rand.Seed(time.Now().UnixNano())
	for _, s := range services {
		go monitorService(s)
	}

	select {}
}
