package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
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

	err = conn.WriteJSON(OpenClientSessionMsg{
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
		Action: Action{
			ActionName:   "xevent",
			Login:        email,
			Password:     hashPassword,
			RequestToken: requestToken,
		},
		CellID: strconv.Itoa(objectID),
		Value:  "0x4001",
		Type:   "HEX",
	})
	if err != nil {
		log.Fatalln("failed to write xevent:", err)
	}

	<-ack
	log.Println("success: xevent")

	err = conn.WriteJSON(Action{
		ActionName:   "get_user_config",
		Login:        email,
		Password:     hashPassword,
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
		var response Response
		err := conn.ReadJSON(&response)
		if err != nil {
			log.Fatalf("failed to read response: %s\n", err)
		}

		fmt.Printf("response to action %s, status %s\n", response.ActionName, response.Status)

		if response.ActionName == "get_user_config" {
			strippedFile := strings.ReplaceAll(response.File, "\\", "")
			fmt.Println("original file:", response.File)
			fmt.Println("stripped file:", strippedFile)

			file := File{}
			err := json.Unmarshal([]byte(strippedFile), &file)
			if err != nil {
				log.Fatalf("failed to unmarshal json: %+v\n", err)
			}

			fmt.Printf("data: %+v\n", file)
		}

		if response.Status == "ok" {
			ack <- struct{}{}
		}
	}
}
