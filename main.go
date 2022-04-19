package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/bartekpacia/fhome/fhome"
	"github.com/gorilla/websocket"
)

var (
	email        string
	passwordHash string
	uniqueID     string
)

var objectID int

const requestToken = "dupadupadupaX"

func init() {
	log.SetFlags(0)
	email = os.Getenv("FHOME_EMAIL")
	passwordHash = os.Getenv("FHOME_PASSWORD_HASH")
	uniqueID = os.Getenv("FHOME_UNIQUE_ID")

	flag.IntVar(&objectID, "object-id", 0, "object id")
	flag.Parse()

	if objectID == 0 {
		log.Fatalln("object-id is required")
	}
}

// There has to be a trailing slash, otherwise handshake fails
const url = "wss://fhome.cloud/webapp-interface/"

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

	err = conn.WriteJSON(fhome.OpenClientToResourceSession{
		ActionName:   fhome.ActionOpenClienToResourceSession,
		Email:        email,
		UniqueID:     uniqueID,
		RequestToken: requestToken,
	})
	if err != nil {
		log.Fatalln("failed to open client to resource session:", err)
	}

	<-ack
	log.Printf("success: open_client_to_resource_session")

	err = conn.WriteJSON(fhome.XEvent{
		ActionName:   fhome.ActionXEvent,
		Login:        email,
		Password:     passwordHash,
		RequestToken: requestToken,
		CellID:       strconv.Itoa(objectID),
		Value:        "0x4001",
		Type:         "HEX",
	})
	if err != nil {
		log.Fatalln("failed to write xevent:", err)
	}

	<-ack
	log.Println("success: xevent")

	err = conn.WriteJSON(fhome.Action{
		ActionName:   fhome.ActionGetUserConfig,
		Login:        email,
		Password:     passwordHash,
		RequestToken: requestToken,
	})
	if err != nil {
		log.Fatalln("failed to write get_user_config:", err)
	}
	<-ack
	<-ack
}

func listen(conn *websocket.Conn, ack chan interface{}) {
	for {
		var response fhome.Response
		err := conn.ReadJSON(&response)
		if err != nil {
			log.Fatalf("failed to read response: %s\n", err)
		}

		fmt.Printf("response to action %s, status %s\n", response.ActionName, response.Status)

		if response.ActionName == "get_user_config" {
			file := fhome.File{}
			err := json.Unmarshal([]byte(response.File), &file)
			if err != nil {
				log.Fatalf("failed to unmarshal json: %+v\n", err)
			}

			fmt.Printf("there are %d cells\n", len(file.Cells))
			for _, cell := range file.Cells {
				fmt.Printf("id: %3d, name: %s\n", cell.ObjectID, cell.Name)
			}
		}

		if response.Status == "ok" {
			ack <- struct{}{}
		}
	}
}
