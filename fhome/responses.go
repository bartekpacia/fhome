package fhome

type Response struct {
	ActionName   string `json:"action_name"`
	RequestToken string `json:"request_token"`
	Status       string `json:"status"`
	Source       string `json:"source"`

	// Non-nil for "disconnecting" action
	Details string `json:"details"`
	Reason  string `json:"reason"`

	// Non-nil for ActionGetUserConfig
	File string `json:"file"`
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
