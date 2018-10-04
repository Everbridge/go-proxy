package admin

import (
	"encoding/json"
	"net/http"

	"github.com/Everbridge/go-proxy/configuration"
	myhttp "github.com/Everbridge/go-proxy/http"
	"github.com/gorilla/mux"
)

func handleGetConfigurations(w http.ResponseWriter, req *http.Request) {
	configs, loadErr := configuration.LoadConfiguration()
	if loadErr != nil {
		myhttp.InternalError(req, w, loadErr)
		return
	}

	responseWithJSON(configs, w, req)
}

func handleSaveConfigurations(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var passedInConfiguration configuration.Configuration
	err := decoder.Decode(&passedInConfiguration)

	if err != nil {
		myhttp.InternalError(req, w, err)
		return
	}

	err = configuration.SaveConfiguration(passedInConfiguration)
	if err != nil {
		myhttp.InternalError(req, w, err)
		return
	}

	responseWithJSON(passedInConfiguration, w, req)
}

func registerConfigurationEndpoints(router *mux.Router) {
	router.HandleFunc("/api/configurations", handleGetConfigurations).Methods(http.MethodGet)
	router.HandleFunc("/api/configurations", handleSaveConfigurations).Methods(http.MethodPut)
}
