package fhome

// Change is a websocket message sent from the server to the client without
// client having to take any action.
//
// Changes are most commonly used to inform the client that the status of some
// resource has changed.

type StatusTouchesChangedResponse struct {
	// TODO: add comments

	ActionName string `json:"action_name"`
	Response   struct {
		ProjectVersion string `json:"ProjectVersion"`
		Status         bool   `json:"Status"`
		StatusText     string `json:"StatusText"`
		Cv             []struct {
			Voi string `json:"VOI"`
			Ii  string `json:"II"`
			Dt  string `json:"DT"`
			Dv  string `json:"DV"`
			Dvs string `json:"DVS"`
		} `json:"CV"`
		ServerTime int `json:"ServerTime"`
	} `json:"response"`
	Status string `json:"status"`
	Source string `json:"source"`
}
