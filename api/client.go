// Package api provides functionality to interact with smart home devices
// connected to F&Home system.
package api

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"strconv"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/pbkdf2"
)

// ErrClientDone is returned when this client can no longer be used.
var ErrClientDone = fmt.Errorf("client is done")

// URL is a URL at which F&Home API lives.
//
// It has to end with a trailing slash, otherwise handshake fails.
const URL = "wss://fhome.cloud/webapp-interface/"

type Client struct {
	email                *string
	resourcePasswordHash *string
	uniqueID             *string

	dialer *websocket.Dialer

	// The first websocket connection that is used for the following actions:
	//  - open_client_session
	//  - get_my_data
	//  - get_my_resources actions
	setupConn *websocket.Conn

	// The second connection that is used for all other actions.
	mainConn *websocket.Conn

	msgStreams map[int]chan<- Message
}

// NewClient returns a new F&Home API client.
//
// If a nil dialer is provided, a default dialer from gorilla/websocket will be
// used.
func NewClient(dialer *websocket.Dialer) (*Client, error) {
	if dialer == nil {
		dialer = websocket.DefaultDialer
	}

	conn, err := connect(dialer)
	if err != nil {
		return nil, fmt.Errorf("connect: %v", err)
	}

	var response Response
	err = conn.ReadJSON(&response)
	if err != nil {
		return nil, fmt.Errorf("read json response: %v", err)
	}

	if response.ActionName != "authentication_required" || response.Status != "" {
		return nil, fmt.Errorf("wrong first message received")
	}

	c := Client{
		email:                nil,
		resourcePasswordHash: nil,
		uniqueID:             nil,
		dialer:               dialer,
		setupConn:            conn,
		mainConn:             nil,
		msgStreams:           make(map[int]chan<- Message),
	}

	return &c, nil
}

// OpenCloudSession opens a websocket connection to F&Home Cloud.
func (c *Client) OpenCloudSession(email, password string) error {
	token := generateRequestToken()

	actionName := ActionOpenClientSession
	err := c.setupConn.WriteJSON(OpenClientSession{
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
		err = c.setupConn.ReadJSON(&response)
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
// this case and assign its unique ID to the client.
func (c *Client) GetMyResources() (*GetMyResourcesResponse, error) {
	token := generateRequestToken()

	actionName := ActionGetMyResources
	err := c.setupConn.WriteJSON(GetMyResources{
		ActionName:   actionName,
		Email:        *c.email,
		RequestToken: token,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to write action %s: %v", actionName, err)
	}

	for {
		var response GetMyResourcesResponse
		err = c.setupConn.ReadJSON(&response)
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
func (c *Client) OpenResourceSession(ctx context.Context, resourcePassword string) error {
	// We can't reuse the connection previously used to connect to Cloud.
	conn, err := connect(c.dialer)
	if err != nil {
		return fmt.Errorf("reconnect: %v", err)
	}

	c.mainConn = conn

	actionName := ActionOpenClienToResourceSession
	token := generateRequestToken()

	err = c.mainConn.WriteJSON(OpenClientToResourceSession{
		ActionName:   actionName,
		Email:        *c.email,
		UniqueID:     *c.uniqueID,
		RequestToken: token,
	})
	if err != nil {
		return fmt.Errorf("failed to write %s: %v", actionName, err)
	}

	go c.reader()

	_, err = c.ReadMessage(ctx, actionName, token)
	if err != nil {
		return fmt.Errorf("failed to read %s: %v", actionName, err)
	}

	c.resourcePasswordHash = generatePasswordHash(resourcePassword)

	return nil
}

// GetSystemConfig returns additional information about particular cells, e.g.,
// their style (icon) and configurator-set name.
//
// Configuration returned by this method is set in the desktop configurator app.
//
// This action is named "Touches" in F&Home's terminology.
func (c *Client) GetSystemConfig(ctx context.Context) (*TouchesResponse, error) {
	actionName := ActionGetSystemConfig
	token := generateRequestToken()

	err := c.mainConn.WriteJSON(Action{
		ActionName:   actionName,
		Login:        *c.email,
		PasswordHash: *c.resourcePasswordHash,
		RequestToken: token,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to write %s: %v", actionName, err)
	}

	msg, err := c.ReadMessage(ctx, actionName, token)
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %v", err)
	}

	var response TouchesResponse
	err = json.Unmarshal(msg.Raw, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %v", err)
	}

	return &response, nil
}

// GetUserConfig returns configuration of cells and panels.
//
// The configuration returned by this method is set in the web or mobile app.
func (c *Client) GetUserConfig(ctx context.Context) (*UserConfig, error) {
	token := generateRequestToken()

	actionName := ActionGetUserConfig
	err := c.mainConn.WriteJSON(Action{
		ActionName:   actionName,
		Login:        *c.email,
		PasswordHash: *c.resourcePasswordHash,
		RequestToken: token,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to write %s to conn: %v", actionName, err)
	}

	msg, err := c.ReadMessage(ctx, actionName, token)
	if err != nil {
		return nil, fmt.Errorf("failed to read messagee: %v", err)
	}

	var userConfigResponse GetUserConfigResponse
	err = json.Unmarshal(msg.Raw, &userConfigResponse)
	if err != nil {
		return nil, fmt.Errorf("unmarshal user config response to json: %v", err)
	}

	var userConfig UserConfig
	err = json.Unmarshal([]byte(userConfigResponse.File), &userConfig)
	if err != nil {
		return nil, fmt.Errorf("unmarshal file to json: %v", err)
	}

	return &userConfig, nil
}

// ReadMessage waits until the client receives a message with matching actionName
// and requestToken.
//
// If requestToken is empty, then it is ignored.
// In such a case, the first message with matching actionName is returned.
//
// If its status is not "ok", it returns an error.
func (c *Client) ReadMessage(ctx context.Context, actionName string, requestToken string) (*Message, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ErrClientDone
		case msg := <-c.read():
			if msg.Status != nil {
				if *msg.Status != "ok" {
					return nil, fmt.Errorf("message status is %s", *msg.Status)
				}
			}

			tokenOk := true
			if requestToken != "" {
				if msg.RequestToken == nil {
					tokenOk = false
				} else if requestToken != *msg.RequestToken {
					tokenOk = false
				}
			}

			if actionName == msg.ActionName && tokenOk {
				return &msg, nil
			}
		}
	}
}

// ReadAnyMessage returns any message received from the server.
//
// If the message has status and it is not ok, it returns an error.
func (c *Client) ReadAnyMessage() (*Message, error) {
	msg := <-c.read()

	if msg.Status != nil {
		if *msg.Status != "ok" {
			return nil, fmt.Errorf("message status is %s", *msg.Status)
		}
	}

	return &msg, nil
}

// SendAction sends an action to the server.
func (c *Client) SendAction(ctx context.Context, actionName string) (*Message, error) {
	token := generateRequestToken()

	action := Action{
		ActionName:   actionName,
		Login:        *c.email,
		PasswordHash: *c.resourcePasswordHash,
		RequestToken: token,
	}

	err := c.mainConn.WriteJSON(action)
	if err != nil {
		return nil, fmt.Errorf("failed to write action %s: %v", action.ActionName, err)
	}

	message, err := c.ReadMessage(ctx, action.ActionName, token)
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %v", err)
	}

	return message, nil
}

// SendEvent sends an event containing value to the cell.
//
// This is a more specific variant of SendAction.
//
// Events are named "Xevents" in F&Home's terminology.
func (c *Client) SendEvent(ctx context.Context, cellID int, value string) error {
	actionName := ActionEvent
	token := generateRequestToken()

	event := Event{
		ActionName:   actionName,
		Login:        *c.email,
		PasswordHash: *c.resourcePasswordHash,
		RequestToken: token,
		CellID:       strconv.Itoa(cellID),
		Value:        value,
		Type:         "HEX",
	}
	err := c.mainConn.WriteJSON(event)
	if err != nil {
		return fmt.Errorf("failed to write %s to conn: %v", actionName, err)
	}

	_, err = c.ReadMessage(ctx, actionName, token)
	if err != nil {
		return fmt.Errorf("failed to read message: %v", err)
	}

	return err
}

func (c *Client) Close() error {
	if err := c.setupConn.Close(); err != nil {
		return fmt.Errorf("failed to close connection 1: %v", err)
	}

	if err := c.mainConn.Close(); err != nil {
		return fmt.Errorf("failed to close connection 2: %v", err)
	}

	return nil
}

func (c *Client) read() <-chan Message {
	msgStream := make(chan Message, 1)
	c.msgStreams[id()] = msgStream
	return msgStream
}

// reader infinitely reads messages from c.conn2 and sends them to all
// subscribers.
func (c *Client) reader() {
	for {
		// read a new message in JSON
		_, data, err := c.mainConn.ReadMessage()
		if err != nil {
			log.Fatalln("failed to read json from conn2:", err)
		}

		// unmarshal it

		var msg Message
		err = json.Unmarshal(data, &msg)
		if err != nil {
			log.Fatalln("failed to unmarshal message:", err)
		}
		msg.Raw = data

		// deliver it to all subscribers
		for id, msgStream := range c.msgStreams {
			msgStream <- msg
			close(msgStream)
			delete(c.msgStreams, id)
		}
	}
}

func connect(dialer *websocket.Dialer) (*websocket.Conn, error) {
	conn, resp, err := dialer.Dial(URL, nil)
	if err != nil {
		if resp != nil {
			log.Println("failed to dial")
			log.Printf("status: %s, headers: %d\n", resp.Status, len(resp.Header))
			for name, value := range resp.Header {
				log.Printf("header %s: %s\n", name, value)
			}
		}

		return nil, fmt.Errorf("failed to dial: %v", err)
	}

	return conn, nil
}

// MergeConfigs creates [Config] config from "get_user_config" action and
// "get_system_config" action.
func MergeConfigs(userConfig *UserConfig, touchesResp *TouchesResponse) (*Config, error) {
	panels := make([]Panel, 0)

	for _, fPanel := range userConfig.Panels {
		uCells := userConfig.GetCellsByPanelID(fPanel.ID)
		cells := make([]Cell, 0)
		for _, fCell := range uCells {
			cell := Cell{
				ID:   fCell.ObjectID,
				Icon: CreateIcon(fCell.Icon),
				Name: fCell.Name,
			}
			cells = append(cells, cell)
		}

		panel := Panel{
			ID:    fPanel.ID,
			Name:  fPanel.Name,
			Cells: cells,
		}

		panels = append(panels, panel)
	}

	cfg := Config{Panels: panels}

	for _, mdcell := range touchesResp.Response.MobileDisplayProperties.Cells {
		cellID, err := strconv.Atoi(mdcell.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to convert cell ID %s to int: %v", mdcell.ID, err)
		}

		cell, err := cfg.GetCellByID(cellID)
		if err != nil {
			// cellID doesn't belong to any panels, ignore this case
			continue
		}

		cell.Desc = mdcell.Desc
		cell.Value = mdcell.Step // FIXME: this is wrong; for thermo-setters this is 0.5, for thermo-getters this is actual value
		cell.TypeNumber = mdcell.TypeNumber
		cell.DisplayType = string(mdcell.DisplayType)
		cell.Preset = mdcell.Preset
		cell.Style = mdcell.Style
		cell.MinValue = mdcell.MinValue
		cell.MaxValue = mdcell.MaxValue
	}

	return &cfg, nil
}

func generateRequestToken() string {
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

// id generates a random int
func id() int {
	return rand.Intn(100000)
}
