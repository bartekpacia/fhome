package fhome

// Message is a websocket message sent from the client to the server.
//
// Messages are most commonly used to make server do perform somehing, e.g
// change the status of some resource/device.

type Message struct {
	ActionName   string  `json:"action_name"`
	RequestToken *string `json:"request_token"`
	Status       *string `json:"status"`
	Orig         []byte
}

const (
	ActionOpenClientSession          = "open_client_session"
	ActionGetMyData                  = "get_my_data"
	ActionGetMyResources             = "get_my_resources"
	ActionOpenClienToResourceSession = "open_client_to_resource_session"
	ActionTouches                    = "touches"
	ActionGetUserConfig              = "get_user_config"
	ActionXEvent                     = "xevent"
	ActionStatusTouchesChanged       = "statustoucheschanged"
)

var ValueToggle = "0x4001"

type OpenClientSession struct {
	ActionName   string `json:"action_name"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	RequestToken string `json:"request_token"`
}

type GetMyResources struct {
	ActionName   string `json:"action_name"`
	Email        string `json:"email"`
	RequestToken string `json:"request_token"`
}

type OpenClientToResourceSession struct {
	ActionName   string `json:"action_name"`
	Email        string `json:"email"`
	UniqueID     string `json:"unique_id"`
	RequestToken string `json:"request_token"`
}

type Action struct {
	ActionName   string `json:"action_name"`
	Login        string `json:"login"`
	PasswordHash string `json:"password"`
	RequestToken string `json:"request_token"`
}

type XEvent struct {
	ActionName   string `json:"action_name"`
	Login        string `json:"login"`
	PasswordHash string `json:"password"`
	RequestToken string `json:"request_token"`
	CellID       string `json:"cell_id"`
	Value        string `json:"value"`
	Type         string `json:"type"`
}
