package main

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

func init() {
	log.SetFlags(0)
}

const url = "wss://fhome.cloud/webapp-interface"

var dialer = websocket.Dialer{
	EnableCompression: true,
	Proxy:             http.ProxyFromEnvironment,
	HandshakeTimeout:  5 * time.Second,
}

func main() {
	headers := http.Header{}
	headers.Add("Pragma", "no-cache")
	headers.Add("Accept-Encoding", "gzip, deflate, br")
	headers.Add("Accept-Language", "pl-PL,pl;q=0.9,en-US;q=0.8,en;q=0.7")
	conn, resp, err := websocket.DefaultDialer.Dial(url, headers)
	if err != nil {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}

		log.Println(resp.Header)
		log.Println("body")
		log.Println(string(body))
		log.Fatalln("failed to dial:", err)
	}

	sessionMsg := make(map[string]string)
	sessionMsg["action_name"] = "open_client_to_resource_session"
	sessionMsg["email"] = "tomekpacia1975@gmail.com"
	sessionMsg["unique_id"] = "uv83fYvi8aFbpe04rhKxfIF7h"
	sessionMsg["request_token"] = "2b359bfa7bb70"

	err = conn.WriteJSON(sessionMsg)
	if err != nil {
		log.Fatalln("failed to write json:", err)
	}

	response := make(map[string]interface{})
	err = conn.ReadJSON(&response)
	if err != nil {
		log.Fatalln("failed to read json:", err)
	}
}
