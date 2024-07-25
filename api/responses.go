package api

import (
	"fmt"
	"log"
	"strconv"
)

// Response is a websocket message sent from the server to the client in
// response to the client's previous websocket message to the server.
type Response struct {
	ActionName   string `json:"action_name"`
	RequestToken string `json:"request_token"`
	Status       string `json:"status"`
	Source       string `json:"source"`

	Details string `json:"details"` // Non-empty for "disconnecting" action
	Reason  string `json:"reason"`  // Non-empty for "disconnecting" action
}

type GetUserConfigResponse struct {
	ActionName   string `json:"action_name"`
	RequestToken string `json:"request_token"`
	Status       string `json:"status"`
	Source       string `json:"source"`
	File         string `json:"file"`
}

type GetMyResourcesResponse struct {
	ActionName    string `json:"action_name"`
	RequestToken  string `json:"request_token"`
	Status        string `json:"status"`
	Source        string `json:"source"`
	AvatarID0     string `json:"avatar_id_0"`
	FriendlyName0 string `json:"friendly_name_0"`
	ResourceType0 string `json:"resource_type_0"`
	UniqueID0     string `json:"unique_id_0"`
}

type TouchesResponse struct {
	ActionName string `json:"action_name"`
	Response   struct {
		ProjectVersion          string `json:"ProjectVersion"`
		Status                  bool   `json:"Status"`
		StatusText              string `json:"StatusText"`
		MobileDisplayProperties struct {
			Cells []MobileDisplayCell `json:"Cells"`
		} `json:"MobileDisplayProperties"`
	} `json:"response"`
	Status       string `json:"status"`
	Source       string `json:"source"`
	RequestToken string `json:"request_token"`
}

type DisplayType string

const (
	Bit         DisplayType = "BIT"
	Byte        DisplayType = "BYTE"
	Temperature DisplayType = "TEMP"
	Percentage  DisplayType = "PROC"
	RGB         DisplayType = "RGB"
)

// MobileDisplayCell is a Cell, but returned from "touches" action.
type MobileDisplayCell struct {
	// Cell description. Note that this is by the configurator app, not by the
	// user in the mobile or web app.
	Desc string `json:"CD"`
	// Object ID
	ID string `json:"OI"`
	// Type number. Known values: 706, 707, 708, 709, 710, 711, 717, 718, 719,
	// 722, 724, 760
	TypeNumber string `json:"TN"`
	// Preset. Known values: 0, 1, 4
	Preset string `json:"P"`
	// Style. Display Type TEMP always has this set to 2.
	Style string `json:"Se"`
	// Minimum value
	MinValue string `json:"Min"`
	// Maximum value
	MaxValue string `json:"Max"`
	// Step (aka current value). Display Type TEMP always has this set to
	// 0xa005.
	//
	// Update 25/07/2024: This is literally the *step*, not current value.
	//
	// To obtain current value, send "touches".
	Step string `json:"Sp"`
	// Display Type.
	DisplayType DisplayType `json:"DT"`
	// Cell permission. Known values: FC (Full Control), RO (Read Only)
	Permission string `json:"CP"`
}

func (cell MobileDisplayCell) String() string {
	return fmt.Sprintf("id: %s, desc: %s, type: %s, preset: %s, style: %s, perm: %s, step/value: %s\n",
		cell.ID, cell.Desc, cell.DisplayType, cell.Preset, cell.Style, cell.Permission, cell.Step,
	)
}

type StatusTouchesChangedResponse struct {
	ActionName string `json:"action_name"`
	Response   struct {
		ProjectVersion string      `json:"ProjectVersion"`
		Status         bool        `json:"Status"`
		StatusText     string      `json:"StatusText"`
		CellValues     []CellValue `json:"CV"`
		ServerTime     int         `json:"ServerTime"`
	} `json:"response"`
	Status string `json:"status"`
	Source string `json:"source"`
}

type CellValue struct {
	ID          string      `json:"VOI"`
	Ii          string      `json:"II"`
	DisplayType DisplayType `json:"DT"`
	Value       string      `json:"DV"`  // Probably "data value"
	ValueStr    string      `json:"DVS"` // Probably "data value string"
}

func (cv *CellValue) IntID() int {
	i, err := strconv.Atoi(cv.ID)
	if err != nil {
		log.Fatalln("failed to convert ID to int:", err)
	}

	return i
}

func (cv CellValue) String() string {
	return fmt.Sprintf("id: %s, ii: %s, dt: %s, dv: %s, dvs: %s",
		cv.ID, cv.Ii, cv.DisplayType, cv.Value, cv.ValueStr,
	)
}
