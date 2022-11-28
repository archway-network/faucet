package faucet

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ignite/cli/ignite/pkg/xhttp"
	"github.com/rs/cors"
)

// ServeHTTP implements http.Handler to expose the functionality of Faucet.Transfer() via HTTP.
// request/response payloads are compatible with the previous implementation at allinbits/cosmos-faucet.
func (f Faucet) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	router := mux.NewRouter()

	router.
		Handle("/", cors.Default().Handler(http.HandlerFunc(f.faucetHandler))).
		Methods(http.MethodPost, http.MethodOptions)

	router.
		HandleFunc("/", OpenAPIHandler("Faucet", "openapi.yml")).
		Methods(http.MethodGet)

	router.
		HandleFunc("/openapi.yml", f.openAPISpecHandler).
		Methods(http.MethodGet)

	router.ServeHTTP(w, r)
}

func (f Faucet) faucetHandler(w http.ResponseWriter, r *http.Request) {
	var req TransferRequest

	// decode request into req.
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responseError(w, http.StatusBadRequest, err)
		return
	}

	// try performing the transfer
	if err := f.Transfer(req); err != nil {
		if err == context.Canceled {
			return
		}
		responseError(w, http.StatusInternalServerError, err)
	} else {
		responseSuccess(w)
	}
}

func responseSuccess(w http.ResponseWriter) {
	xhttp.ResponseJSON(w, http.StatusOK, TransferResponse{})
}

func responseError(w http.ResponseWriter, code int, err error) {
	xhttp.ResponseJSON(w, code, TransferResponse{
		Error: err.Error(),
	})
}

func (f Faucet) openAPISpecHandler(w http.ResponseWriter, r *http.Request) {
	tmplOpenAPISpec.Execute(w, f.AppChainID)
}

// Handler returns an http handler that servers OpenAPI console for an OpenAPI spec at specURL.
func OpenAPIHandler(title, specURL string) http.HandlerFunc {
	t, _ := template.ParseFS(index, "openapi/index.tpl")

	return func(w http.ResponseWriter, req *http.Request) {
		t.Execute(w, struct {
			Title string
			URL   string
		}{
			title,
			specURL,
		})
	}
}
