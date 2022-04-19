package fhome

const (
	ActionOpenClientSession          = "open_client_session"
	ActionGetMyData                  = "get_my_data"
	ActionGetMyResources             = "get_my_resources"
	ActionOpenClienToResourceSession = "open_client_to_resource_session"
	ActionTouches                    = "touches"
	ActionGetUserConfig              = "get_user_config"
	ActionXEvent                     = "xevent"
)

type OpenClientSession struct {
	ActionName   string `json:"action_name"`
	Email        string `json:"email"`
	Password     string `json:"password"`
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
	Password     string `json:"password"`
	RequestToken string `json:"request_token"`
}

type XEvent struct {
	ActionName   string `json:"action_name"`
	Login        string `json:"login"`
	Password     string `json:"password"`
	RequestToken string `json:"request_token"`
	CellID       string `json:"cell_id"`
	Value        string `json:"value"`
	Type         string `json:"type"`
}
