package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

var (
	email        string
	password     string
	hashPassword string
	uniqueID     string
)

var objectID int

const requestToken = "dupadupadupaX"

func init() {
	log.SetFlags(0)
	email = os.Getenv("FHOME_EMAIL")
	password = os.Getenv("FHOME_PASSWORD")
	hashPassword = os.Getenv("FHOME_HASH_PASSWORD")
	uniqueID = os.Getenv("FHOME_UNIQUE_ID")

	flag.IntVar(&objectID, "object-id", 0, "object id")
	flag.Parse()

	if objectID == 0 {
		log.Fatalln("object-id is required")
	}
}

const url = "wss://fhome.cloud/webapp-interface/" // There has to be a trailing slash, otherwise handshake fails

var dialer = websocket.Dialer{
	EnableCompression: true,
	HandshakeTimeout:  5 * time.Second,
}

func main() {
	conn, resp, err := dialer.Dial(url, nil)
	if err != nil {
		log.Printf("status: %s\n", resp.Status)
		for name, value := range resp.Header {
			log.Printf("header %s: %s\n", name, value)
		}

		log.Fatalln("failed to dial:", err)
	}
	defer conn.Close()

	ack := make(chan interface{})
	go listen(conn, ack)

	err = conn.WriteJSON(OpenClientSessionsMsg{
		ActionName:   "open_client_to_resource_session",
		Email:        email,
		UniqueID:     uniqueID,
		RequestToken: requestToken,
	})
	if err != nil {
		log.Fatalln("failed to open client to resource session:", err)
	}

	<-ack
	log.Printf("success: open_client_to_resource_session")

	err = conn.WriteJSON(XEventMsg{
		ActionName:   "xevent",
		CellID:       strconv.Itoa(objectID),
		Value:        "0x4001",
		Type:         "HEX",
		Login:        email,
		Password:     hashPassword,
		RequestToken: requestToken,
	})
	if err != nil {
		log.Fatalln("failed to write xevent:", err)
	}

	<-ack
	log.Println("success: xevent")
}

func listen(conn *websocket.Conn, ack chan interface{}) error {
	for {
		var response Response
		err := conn.ReadJSON(&response)
		if err != nil {
			return fmt.Errorf("failed to read json: %v", err)
		}

		fmt.Printf("response: %+v\n", response)

		if response.Status == "ok" {
			ack <- struct{}{}
		}
	}
}
