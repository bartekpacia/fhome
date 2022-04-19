package fhome

type Response struct {
	ActionName   string `json:"action_name"`
	RequestToken string `json:"request_token"`
	Status       string `json:"status"`
	Source       string `json:"source"`
	File         string `json:"file"`
}
