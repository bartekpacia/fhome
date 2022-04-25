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

	messages chan Message
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

// OpenCloudSession opens a websocket connection to F&Home Cloud.
func (c *Client) OpenCloudSession(email, password string) error {
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
			return fmt.Errorf("response status is %#v", response.Status)
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

// OpenResourceSession open a websocket connection to the only resource that the
// user has.
//
// Currently, it assumes that a user has only one resource.
func (c *Client) OpenResourceSession(resourcePassword string) error {
	// We can't use the connection that was used to connect to Cloud.
	conn, err := connect()
	if err != nil {
		return fmt.Errorf("reconnect: %v", err)
	}

	c.conn2 = conn
	c.messages = make(chan Message)

	actionName := ActionOpenClienToResourceSession
	token := generateRequestToken()

	err = c.conn2.WriteJSON(OpenClientToResourceSession{
		ActionName:   actionName,
		Email:        *c.email,
		UniqueID:     *c.uniqueID,
		RequestToken: token,
	})
	if err != nil {
		return fmt.Errorf("failed to write %s: %v", actionName, err)
	}

	go c.msgReader() // TODO: think about closing this goroutine

	_, err = c.readMsg(&actionName, &token)
	if err != nil {
		return fmt.Errorf("failed to read %s: %v", actionName, err)
	}

	c.resourcePasswordHash = generatePasswordHash(resourcePassword)

	return nil
}

// readMsg waits until the client receives message with matching actionName and
// requestToken.
//
// If actionName or requestToken is null, then it is ignored.
//
// If its status is not "ok", it returns an error.
func (c *Client) readMsg(actionName *string, requestToken *string) (*Message, error) {
	for {
		msg := <-c.messages

		if msg.Status != nil {
			if *msg.Status != "ok" {
				return nil, fmt.Errorf("message status is %s", *msg.Status)
			}
		}

		actionOk := true
		if actionName != nil {
			if *actionName != msg.ActionName {
				actionOk = false
			}
		}

		tokenOk := true
		if requestToken != nil {
			if msg.RequestToken == nil {
				tokenOk = false
			} else if *requestToken != *msg.RequestToken {
				tokenOk = false
			}
		}

		if actionOk && tokenOk {
			return &msg, nil
		}
	}
}

func (c *Client) msgReader() {
	for {
		_, msgByte, err := c.conn2.ReadMessage()
		if err != nil {
			log.Fatalln("failed to read json from conn2:", err)
		}

		var msg Message
		err = json.Unmarshal(msgByte, &msg)
		if err != nil {
			log.Fatalln("failed to unmarshal message:", err)
		}

		msg.Orig = msgByte

		c.messages <- msg
	}
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

	msg, err := c.readMsg(&actionName, &token)
	if err != nil {
		return nil, fmt.Errorf("failed to read messagee: %v", err)
	}

	var userConfigResponse GetUserConfigResponse
	err = json.Unmarshal(msg.Orig, &userConfigResponse)
	if err != nil {
		return nil, fmt.Errorf("unmarshal user config response to json: %v", err)
	}

	var file File
	err = json.Unmarshal([]byte(userConfigResponse.File), &file)
	if err != nil {
		return nil, fmt.Errorf("unmarshal file to json: %v", err)
	}

	return &file, nil
}

func (c *Client) XEvent(resourceID int, value string) error {
	actionName := ActionXEvent
	token := generateRequestToken()

	xevent := XEvent{
		ActionName:   actionName,
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

	_, err = c.readMsg(&actionName, &token)
	return err
}

func (c *Client) Listen(messages chan Message, errors chan error) {
	messagesInternal := make(chan Message)
	errorsInternal := make(chan error)

	listener := func() {
		for {
			msg, err := c.readMsg(nil, nil)
			if err != nil {
				errorsInternal <- err
			}

			messagesInternal <- *msg
		}
	}

	go listener()

	for {
		select {
		case response := <-messagesInternal:
			messages <- response
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
