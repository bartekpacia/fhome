package webserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bartekpacia/fhome/api"
)

type Api struct {
	fhomeClient *api.Client
}

func NewApi(fhomeClient *api.Client) *Api {
	return &Api{
		fhomeClient: fhomeClient,
	}
}

func (a *Api) Mux() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /devices", a.getDevices)
	mux.HandleFunc("POST /devices/{id}", a.toggleDevice)

	authMux := withPassphrase(mux, "my-passphrase")
	return authMux
}

func (a *Api) getDevices(w http.ResponseWriter, r *http.Request) {
	userConfig, err := a.fhomeClient.GetUserConfig(r.Context())
	if err != nil {
		http.Error(w, "failed to get user config"+err.Error(), http.StatusInternalServerError)
		return
	}

	response := make([]device, 0)

	for _, cell := range userConfig.Cells {
		response = append(response, device{
			Name: cell.Name,
			ID:   cell.ObjectID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "failed to encode user into json"+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (a *Api) toggleDevice(w http.ResponseWriter, r *http.Request) {
	objectID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = a.fhomeClient.SendEvent(r.Context(), int(objectID), api.ValueToggle)
	if err != nil {
		msg := fmt.Sprintf("failed to send event to object with %d: %v\n", objectID, err)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
}

func withPassphrase(next http.Handler, passphrase string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Passphrase: "+passphrase {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type device struct {
	Name string
	ID   int
}
