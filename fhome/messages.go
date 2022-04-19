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

type XEventMsg struct {
	Action
	CellID string `json:"cell_id"`
	Value  string `json:"value"`
	Type   string `json:"type"`
}

type Response struct {
	ActionName string `json:"action_name"`
	Status     string `json:"status"`
	Source     string `json:"source"`
	File       string `json:"file"`
}

type File struct {
	Cells  []Cell  `json:"cells"`
	Panels []Panel `json:"panels"`
	Server struct {
		ProjectVersion string `json:"projectVersion"`
	} `json:"server"`
}

type Cell struct {
	ObjectID        int               `json:"objectId"`
	Icon            string            `json:"icon"`
	Name            string            `json:"name"`
	PositionInPanel []PositionInPanel `json:"positionInPanel"`
}

type PositionInPanel struct {
	Orientation string `json:"orientation"`
	PanelID     string `json:"panelId"`
	X           int    `json:"x"`
	Y           int    `json:"y"`
}

type Panel struct {
	ID                   string      `json:"id"`
	Name                 string      `json:"name"`
	X                    int         `json:"x"`
	Y                    int         `json:"y"`
	Icon                 interface{} `json:"icon"`
	ColumnCountPortrait  int         `json:"columnCountPortrait"`
	ColumnCountLandscape int         `json:"columnCountLandscape"`
}
