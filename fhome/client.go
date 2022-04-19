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

	OpenClientToResourceSession(resourceID string) error

	GetUserConfig() (*File, error)

	XEvent(resourceID string, value string, eventType string) error
}

type client struct {
	conn     *websocket.Conn
	email    *string
	password *string // hashed
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
		c.password = &password // FIXME: this is unqiueID, not a real password

		fmt.Printf("response to action %s, status %s\n", response.ActionName, response.Status)

		return nil

	}
}

func (c *client) OpenClientToResourceSession(resourceID string) error {
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
