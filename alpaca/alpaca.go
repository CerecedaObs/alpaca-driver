package alpaca

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
)

// Global transaction counter
var txCounter atomic.Int32

type baseResponse struct {
	ClientTransactionID int    `json:"ClientTransactionID"`
	ServerTransactionID int    `json:"ServerTransactionID"`
	ErrorNumber         int    `json:"ErrorNumber"`
	ErrorMessage        string `json:"ErrorMessage"`
	Value               any    `json:"Value,omitempty"`
}

// Helper to read and parse the request body as URL-encoded data.
func parseBodyParams(r *http.Request) (url.Values, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	// Reset the body so it can be read again later.
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	return url.ParseQuery(string(bodyBytes))
}

// getClientTxID obtains the client transaction ID from the request body.
func getClientTxID(params url.Values, path string) (int, error) {
	if strings.HasPrefix(path, "/management") {
		return 0, nil
	}

	for param, value := range params {
		if strings.ToLower(param) == "clienttransactionid" {
			id, _ := strconv.Atoi(value[0])
			if id < 0 {
				return 0, errors.New("ClientTransactionID must be non-negative")
			}
			return id, nil
		}
	}
	return 0, errors.New("missing ClientTransactionID")
	// return 0, nil
}

func handleResponse(w http.ResponseWriter, r *http.Request, value any) {
	var params url.Values

	if r.Method == "PUT" {
		// PUT requests have the parameters in the body.
		params, _ = parseBodyParams(r)

	} else {
		// GET requests have the parameters in the URL.
		params = r.URL.Query()
	}

	txID, err := getClientTxID(params, r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := baseResponse{
		ServerTransactionID: int(txCounter.Add(1)),
		ClientTransactionID: txID,
	}
	if value != nil {
		response.Value = value
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleError(w http.ResponseWriter, r *http.Request, code int, message string) {
	params := r.URL.Query()

	txID, err := getClientTxID(params, r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response := baseResponse{
		ServerTransactionID: int(txCounter.Add(1)),
		ClientTransactionID: txID,
		ErrorNumber:         code,
		ErrorMessage:        message,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// parseRequest now reads the field from the request body.
func parseRequest(r *http.Request, field string) (string, error) {
	params, err := parseBodyParams(r)
	if err != nil {
		return "", err
	}

	value, ok := params[field]
	if !ok {
		return "", errors.New("missing field")
	}
	return value[0], nil
}

func parseBoolRequest(r *http.Request, field string) (bool, error) {
	value, err := parseRequest(r, field)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(value)
}

func parseFloatRequest(r *http.Request, field string) (float64, error) {
	value, err := parseRequest(r, field)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(value, 64)
}
