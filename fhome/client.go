// Package fhome provides functionality to interact with smart home devices
// connected to F&Home system.
package fhome

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/pbkdf2"
)

// URL at which F&Home API lives.
//
// It has to end with a trailing slash, otherwise handshake fails.
const apiURL = "wss://fhome.cloud/webapp-interface/"

var dialer = websocket.Dialer{
	EnableCompression: true,
	HandshakeTimeout:  5 * time.Second,
}

type Client struct {
	// First websocket connection that is used for open_client_session,
	// get_my_data and get_my_resources actions.
	conn1                *websocket.Conn
	conn2                *websocket.Conn
	email                *string
	resourcePasswordHash *string
	uniqueID             *string
}

// NewClient creates a new client and automatically starts connecting to
// websockets.
func NewClient() (*Client, error) {
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

	c := Client{conn1: conn}
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

func (c *Client) Close() error {
	if err := c.conn1.Close(); err != nil {
		return fmt.Errorf("failed to close connection 1: %v", err)
	}

	if err := c.conn2.Close(); err != nil {
		return fmt.Errorf("failed to close connection 2: %v", err)
	}

	return nil
}

func (c *Client) OpenClientSession(email, password string) error {
	token := generateRequestToken()

	actionName := ActionOpenClientSession
	err := c.conn1.WriteJSON(OpenClientSession{
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
		err = c.conn1.ReadJSON(&response)
		if err != nil {
			return fmt.Errorf("failed to read response: %v", err)
		}

		if response.Status != "ok" {
			return fmt.Errorf("response status is %s", response.Status)
		}

		if response.RequestToken != token || response.ActionName != actionName {
			continue
		}

		c.email = &email
		// c.resourcePasswordHash = generatePasswordHash(password)

		return nil
	}
}

// GetMyResources gets resources assigned to the user.
//
// Most of the time, there will be just one resource. Currently we handle only
// this case and assign its unique ID on the client.
func (c *Client) GetMyResources() (*GetMyResourcesResponse, error) {
	token := generateRequestToken()

	actionName := ActionGetMyResources
	err := c.conn1.WriteJSON(GetMyResources{
		ActionName:   actionName,
		Email:        *c.email,
		RequestToken: token,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to write action %s: %v", actionName, err)
	}

	for {
		var response GetMyResourcesResponse
		err = c.conn1.ReadJSON(&response)
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

		return &response, nil
	}
}

// OpenClientToResourceSession connects to the user's resource.
//
// Currently, it assumes that a user has only one resource.
func (c *Client) OpenClientToResourceSession(resourcePassword string) error {
	// We can't use the old connection.
	conn, err := connect()
	if err != nil {
		return fmt.Errorf("reconnect: %v", err)
	}

	c.conn2 = conn

	token := generateRequestToken()

	actionName := ActionOpenClienToResourceSession
	err = c.conn2.WriteJSON(OpenClientToResourceSession{
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
		err = c.conn2.ReadJSON(&response)
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

		break
	}

	c.resourcePasswordHash = generatePasswordHash(resourcePassword)

	return nil
}

func (c *Client) GetUserConfig() (*File, error) {
	token := generateRequestToken()

	actionName := ActionGetUserConfig
	err := c.conn2.WriteJSON(Action{
		ActionName:   actionName,
		Login:        *c.email,
		PasswordHash: *c.resourcePasswordHash,
		RequestToken: token,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to write %s to conn: %v", actionName, err)
	}

	for {
		var response Response
		err = c.conn2.ReadJSON(&response)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %v", err)
		}

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

		return &file, nil
	}
}

func (c *Client) XEvent(resourceID int, value string) error {
	token := generateRequestToken()

	actionName := ActionXEvent
	xevent := XEvent{
		ActionName:   ActionXEvent,
		Login:        *c.email,
		PasswordHash: *c.resourcePasswordHash,
		RequestToken: token,
		CellID:       strconv.Itoa(resourceID),
		Value:        value,
		Type:         "HEX",
	}
	err := c.conn2.WriteJSON(xevent)
	if err != nil {
		return fmt.Errorf("failed to write %s to conn: %v", actionName, err)
	}

	for {
		var response Response
		err = c.conn2.ReadJSON(&response)
		if err != nil {
			return fmt.Errorf("failed to read response: %v", err)
		}

		if response.RequestToken != token {
			continue
		}

		if response.Status != "ok" || response.ActionName != actionName {
			continue
		}

		return nil

	}
}

func (c *Client) Listen(responses chan StatusTouchesChangedResponse, errors chan error) {
	responsesInternal := make(chan StatusTouchesChangedResponse)
	errorsInternal := make(chan error)

	listener := func() {
		for {
			msgType, msg, err := c.conn2.ReadMessage()
			if err != nil {
				errorsInternal <- fmt.Errorf("read message from conn2: %v", err)
				return
			}

			fmt.Println("new msg: msgType:", msgType, "content:", string(msg))

			var response StatusTouchesChangedResponse
			err = json.Unmarshal(msg, &response)
			if err != nil {
				errorsInternal <- fmt.Errorf("unmarshal message into json: %v", err)
				return
			}

			responsesInternal <- response
		}
	}

	go listener()

	for {
		select {
		case response := <-responsesInternal:
			responses <- response
		case err := <-errorsInternal:
			errors <- err
		}
	}
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

func generatePasswordHash(password string) *string {
	const wordSizeInBytes = 4
	const salt = "fhome123" // yes, they really did it

	keyLength := (256 / 32) * wordSizeInBytes

	hash := pbkdf2.Key([]byte(password), []byte(salt), 1e4, keyLength, sha1.New)
	stringHash := base64.StdEncoding.EncodeToString(hash)
	return &stringHash
}
