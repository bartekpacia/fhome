package api

import "encoding/json"

// Pprint pretty prints i.
func Pprint(i any) string {
	s, _ := json.MarshalIndent(i, "", "\t")
	return string(s)
}
