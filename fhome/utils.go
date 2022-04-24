package fhome

import "encoding/json"

// Pprint pretty prints i.
func Pprint(i interface{}) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}
