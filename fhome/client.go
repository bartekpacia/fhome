// Package fhome provides functionality to interact with smart home devices
// connected to F&Home system.
package fhome

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/gorilla/websocket"
)

// URL at which F&Home API lives.
//
// It has to end with a trailing slash, otherwise handshake fails.
const apiURL = "wss://fhome.cloud/webapp-interface/"

var dialer = websocket.Dialer{
	EnableCompression: true,
	HandshakeTimeout:  5 * time.Second,
}

type Client interface {
	Close() error

	OpenClientSession(email, password string) error

	GetMyResources() (*GetMyResourcesResponse, error)

	OpenClientToResourceSession() error

	GetUserConfig() (*File, error)

	XEvent(resourceID string, value string, eventType string) error
}

type client struct {
	conn     *websocket.Conn
	email    *string
	uniqueID *string
}

// NewClient creates a new client and automatically starts connecting to
// websockets.
func NewClient() (Client, error) {
	conn, err := connect()
	if err != nil {
		return nil, fmt.Errorf("create client: %v", err)
	}

	var response Response
	err = conn.ReadJSON(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to read initial response")
	}

	if response.ActionName != "authentication_required" || response.Status != "" {
		return nil, fmt.Errorf("wrong first message received")
	}

	c := client{conn: conn}
	return &c, nil
}

func connect() (*websocket.Conn, error) {
	conn, resp, err := dialer.Dial(apiURL, nil)
	if err != nil {
		log.Printf("status: %s\n", resp.Status)
		for name, value := range resp.Header {
			log.Printf("header %s: %s\n", name, value)
		}

		return nil, fmt.Errorf("failed to dial: %v", err)
	}

	return conn, nil
}

func (c *client) Close() error {
	err := c.conn.Close()
	if err != nil {
		return fmt.Errorf("failed to close connection: %v", err)
	}

	return nil
}

func (c *client) OpenClientSession(email, password string) error {
	token := generateRequestToken()

	actionName := ActionOpenClientSession
	err := c.conn.WriteJSON(OpenClientSession{
		ActionName:   actionName,
		Email:        email,
		Password:     password,
		RequestToken: token,
	})
	if err != nil {
		return fmt.Errorf("failed to write %s: %v", actionName, err)
	}

	for {
		var response Response
		err = c.conn.ReadJSON(&response)
		if err != nil {
			return fmt.Errorf("failed to read response: %v", err)
		}

		log.Println("xd")

		if response.Status != "ok" {
			return fmt.Errorf("response status is %s", response.Status)
		}

		if response.RequestToken != token || response.ActionName != actionName {
			continue
		}

		c.email = &email

		fmt.Printf("response to action %s, status %s\n", response.ActionName, response.Status)

		return nil
	}
}

// Gets resources assigned to the user.
//
// Most of the time, there will be just one resource. Currently we handle only
// this case and assign its unique ID on the client.
func (c *client) GetMyResources() (*GetMyResourcesResponse, error) {
	token := generateRequestToken()

	actionName := ActionGetMyResources
	err := c.conn.WriteJSON(GetMyResources{
		ActionName:   actionName,
		Email:        *c.email,
		RequestToken: token,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to write action %s: %v", actionName, err)
	}

	for {
		var response GetMyResourcesResponse
		err = c.conn.ReadJSON(&response)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %v", err)
		}

		if response.Status != "ok" {
			return nil, fmt.Errorf("response status is %s", response.Status)
		}

		if response.RequestToken != token || response.ActionName != actionName {
			continue
		}

		c.uniqueID = &response.UniqueID0

		fmt.Printf("response to action %s, status %s\n", response.ActionName, response.Status)

		return &response, nil
	}
}

// Connects to the resource of id c.uniqueID.
func (c *client) OpenClientToResourceSession() error {
	// we have to reconnect
	conn, err := connect()
	if err != nil {
		return fmt.Errorf("reconnect: %v", err)
	}

	c.conn = conn

	token := generateRequestToken()

	fmt.Println("email:", *c.email, ", uniqueID:", *c.uniqueID, ", token:", token)

	actionName := ActionOpenClienToResourceSession
	err = c.conn.WriteJSON(OpenClientToResourceSession{
		ActionName:   actionName,
		Email:        *c.email,
		UniqueID:     *c.uniqueID,
		RequestToken: token,
	})
	if err != nil {
		return fmt.Errorf("failed to write %s: %v", actionName, err)
	}

	for {
		var response Response
		err = c.conn.ReadJSON(&response)
		if err != nil {
			return fmt.Errorf("failed to read response: %v", err)
		}

		if response.RequestToken != token {
			continue
		}

		if response.Status != "ok" {
			fmt.Printf("response: %+v\n", response)
			return fmt.Errorf("response status is %s", response.Status)
		}

		fmt.Printf("response to action %s, status %s\n", response.ActionName, response.Status)

		return nil
	}
}

func (c *client) GetUserConfig() (*File, error) {
	token := generateRequestToken()

	actionName := ActionGetUserConfig
	err := c.conn.WriteJSON(Action{
		ActionName:   actionName,
		Login:        *c.email,
		Password:     *c.uniqueID, // FIXME: This has to be some hashed password
		RequestToken: token,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to write %s to conn: %v", actionName, err)
	}

	for {
		var response Response
		err = c.conn.ReadJSON(&response)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %v", err)
		}
		log.Printf("response: %+v\n", response)

		if response.RequestToken != token {
			continue
		}

		if response.Status != "ok" || response.ActionName != actionName {
			continue
		}

		file := File{}
		err = json.Unmarshal([]byte(response.File), &file)
		if err != nil {
			return nil, fmt.Errorf("unmarshal json: %v", err)
		}

		fmt.Printf("there are %d cells\n", len(file.Cells))
		for _, cell := range file.Cells {
			fmt.Printf("id: %3d, name: %s\n", cell.ObjectID, cell.Name)
		}

		return &file, nil
	}
}

func (c *client) XEvent(resourceID string, value string, eventType string) error {
	return fmt.Errorf("not implemented yet")
}

func generateRequestToken() string {
	rand.Seed(time.Now().UnixNano())
	letters := []rune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz")

	b := make([]rune, 13)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}
