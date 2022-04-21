package fhome

import "strconv"

const (
	ActionOpenClientSession          = "open_client_session"
	ActionGetMyData                  = "get_my_data"
	ActionGetMyResources             = "get_my_resources"
	ActionOpenClienToResourceSession = "open_client_to_resource_session"
	ActionTouches                    = "touches"
	ActionGetUserConfig              = "get_user_config"
	ActionXEvent                     = "xevent"
)

var (
	ValueToggle = strconv.FormatInt(0x4001, 8)
	Value0      = strconv.FormatInt(0x6000, 8)
	Value20     = strconv.FormatInt(0x6014, 8)
	Value40     = strconv.FormatInt(0x6028, 8)
	Value60     = strconv.FormatInt(0x603C, 8)
	Value80     = strconv.FormatInt(0x6050, 8)
	Value100    = strconv.FormatInt(0x6064, 8)
)

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
