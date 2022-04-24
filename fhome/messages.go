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

var (
	ValueToggle = "0x4001"
	Value0      = "0x6000"
	Value20     = "0x6014"
	Value40     = "0x6028"
	Value60     = "0x603C"
	Value80     = "0x6050"
	Value100    = "0x6064"
)

func MapToValue(v int) string {
	switch value := v; {
	case value < 0:
		return Value0
	case value < 20:
		return Value20
	case value < 40:
		return Value40
	case value < 60:
		return Value60
	case value < 80:
		return Value80
	case value < 100:
		return Value100
	default:
		return Value100
	}
}

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
