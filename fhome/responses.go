package fhome

type Response struct {
	ActionName   string `json:"action_name"`
	RequestToken string `json:"request_token"`
	Status       string `json:"status"`
	Source       string `json:"source"`
	File         string `json:"file"`
}

type GetMyResourcesResponse struct {
	ActionName    string `json:"action_name"`
	AvatarID0     string `json:"avatar_id_0"`
	FriendlyName0 string `json:"friendly_name_0"`
	RequestToken  string `json:"request_token"`
	ResourceType0 string `json:"resource_type_0"`
	Source        string `json:"source"`
	Status        string `json:"status"`
	UniqueID0     string `json:"unique_id_0"`
}
