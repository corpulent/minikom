package main

import (
	"bytes"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

func getenv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("Missing environment variable: %s\n", key)
		return ""
	}
	return v
}

func delay(condition bool, maxSeconds int) {
	if condition {
		time.Sleep(time.Duration(rand.Intn(maxSeconds)) * time.Second)
	}
}

func postJSON(endpoint string, data map[string]interface{}) {
	buf, _ := json.Marshal(data)
	_, err := http.Post(endpoint, "application/json", bytes.NewReader(buf))
	if err != nil {
		log.Fatalln(err)
	}
}
