package main

import (
	"flag"
	"io"
	"log"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

var (
	email    string
	uniqueID string
)

func init() {
	log.SetFlags(0)

	flag.StringVar(&email, "email", "", "email")
	flag.StringVar(&uniqueID, "unique-id", "", "unique id")
}

const url = "wss://fhome.cloud/webapp-interface/" // There has to be a trailing slash, otherwise handshake fails

var dialer = websocket.Dialer{
	EnableCompression: true,
	HandshakeTimeout:  5 * time.Second,
}

func main() {
	// headers := http.Header{} headers.Add("Pragma", "no-cache")
	// headers.Add("Accept-Encoding", "gzip, deflate, br")
	// headers.Add("Accept-Language", "pl-PL,pl;q=0.9,en-US;q=0.8,en;q=0.7")
	conn, resp, err := dialer.Dial(url, nil)
	if err != nil {
		log.Println("failed to dial:", err)

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln("failed to read body:", err)
		}

		log.Printf("header: %s\n", resp.Header)
		log.Printf("body: %s\n", string(body))

		os.Exit(1)
	}
	defer conn.Close()

	sessionMsg := make(map[string]string)
	sessionMsg["action_name"] = "open_client_to_resource_session"
	sessionMsg["email"] = email
	sessionMsg["unique_id"] = uniqueID
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

	log.Println("success")
}
