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

type TouchesResponse struct {
	ActionName string `json:"action_name"`
	Response   struct {
		ProjectVersion          string `json:"ProjectVersion"`
		Status                  bool   `json:"Status"`
		StatusText              string `json:"StatusText"`
		MobileDisplayProperties struct {
			Cells []MobileDisplayCell `json:"Cells"`
		} `json:"MobileDisplayProperties"`
	} `json:"response"`
	Status       string `json:"status"`
	Source       string `json:"source"`
	RequestToken string `json:"request_token"`
}

// MobileDisplayCell is a Cell, but returned from "touches" action.
type MobileDisplayCell struct {
	// Cell description
	Cd string `json:"CD"`
	// Object ID
	Oi string `json:"OI"`
	// Type number. Known values: 706, 707, 708, 709, 710, 711, 717, 718, 719,
	// 722, 724, 760
	Tn string `json:"TN"`
	// Preset. Known values: 0, 1, 4
	P string `json:"P"`
	// Style
	Se  string `json:"Se"`
	Min string `json:"Min"`
	Max string `json:"Max"`
	// Step (aka current value).
	Sp string `json:"Sp"`
	// Display Type. Known values: BIT, BYTE, TEMP (Temperature), PROC
	// (Percentage), RGB (Light)
	Dt string `json:"DT"`
	// Cell permission. Known values: FC (Full Control), RO (Read Only)
	Cp string `json:"CP"`
}
