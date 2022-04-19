package main

type OpenClientSessionsMsg struct {
	ActionName   string `json:"action_name"`
	Email        string `json:"email"`
	UniqueID     string `json:"unique_id"`
	RequestToken string `json:"request_token"`
}

type XEventMsg struct {
	ActionName   string `json:"action_name"`
	CellID       string `json:"cell_id"`
	Value        string `json:"value"`
	Type         string `json:"type"`
	Login        string `json:"login"`
	Password     string `json:"password"`
	RequestToken string `json:"request_token"`
}

type Response struct {
	ActionName string `json:"action_name"`
	Status     string `json:"status"`
	Source     string `json:"source"`
}
