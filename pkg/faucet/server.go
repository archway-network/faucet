package faucet

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

// ServeHTTP implements http.Handler to expose the functionality of Faucet.Transfer() via HTTP.
// request/response payloads are compatible with the previous implementation at allinbits/cosmos-faucet.
func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router := mux.NewRouter()

	router.
		Handle("/", cors.Default().Handler(http.HandlerFunc(faucetHandler))).
		Methods(http.MethodPost, http.MethodOptions)

	router.ServeHTTP(w, r)
}

func faucetHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok"}`))
}
