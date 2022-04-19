// Package fhome provides functionality to interact with smart home devices
// connected to F&Home system.
package fhome

import (
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
	Connect() error

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

func NewClient() Client {
	return &client{}
}

func (c *client) Connect() error {
	conn, resp, err := dialer.Dial(apiURL, nil)
	if err != nil {
		log.Printf("status: %s\n", resp.Status)
		for name, value := range resp.Header {
			log.Printf("header %s: %s\n", name, value)
		}

		return fmt.Errorf("failed to dial: %v", err)
	}

	c.conn = conn
	return nil
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

	err := c.conn.WriteJSON(OpenClientSession{
		ActionName:   ActionOpenClientSession,
		Email:        email,
		Password:     password,
		RequestToken: token,
	})
	if err != nil {
		return fmt.Errorf("failed to open client session: %v", err)
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

	err := c.conn.WriteJSON(GetMyResources{
		ActionName:   ActionGetMyResources,
		Email:        *c.email,
		RequestToken: token,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get my resoucres: %v", err)
	}

	for {
		var response GetMyResourcesResponse
		err = c.conn.ReadJSON(&response)
		if err != nil {
			return nil, fmt.Errorf("failed to read response: %v", err)
		}

		if response.RequestToken != token {
			continue
		}

		c.uniqueID = &response.UniqueID0

		fmt.Printf("response to action %s, status %s\n", response.ActionName, response.Status)

		return &response, nil
	}
}

// Connects to the resource of id c.uniqueID.
func (c *client) OpenClientToResourceSession() error {
	token := generateRequestToken()

	err := c.conn.WriteJSON(OpenClientToResourceSession{
		ActionName:   ActionOpenClienToResourceSession,
		Email:        *c.email,
		UniqueID:     *c.uniqueID,
		RequestToken: token,
	})
	if err != nil {
		return fmt.Errorf("failed to open client to resource session: %v", err)
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

		fmt.Printf("response to action %s, status %s\n", response.ActionName, response.Status)

		return nil
	}
}

func (c *client) GetUserConfig() (*File, error) {
	return nil, fmt.Errorf("not implemented yet")
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
