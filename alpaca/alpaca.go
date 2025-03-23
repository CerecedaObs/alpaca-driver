package alpaca

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
)

var (
	errBadRequest = errors.New("bad request")
	errInternal   = errors.New("internal error")
)

// Global transaction counter
var txCounter atomic.Int32

type baseResponse struct {
	ClientTransactionID int    `json:"ClientTransactionID"`
	ServerTransactionID int    `json:"ServerTransactionID"`
	ErrorNumber         int    `json:"ErrorNumber,omitempty"`
	ErrorMessage        string `json:"ErrorMessage,omitempty"`
	Value               any    `json:"Value,omitempty"`
}

// Define a custom type for context keys
type contextKey string

const paramsKey contextKey = "params"

// handleMgm wraps a management handler function and returns an http.Handler.
// Management handlers do not require a ClientTransactionID.
func handleMgm(handler func(r *http.Request) (any, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var response baseResponse

		value, err := handler(r)
		if err != nil {
			// TODO: Define error numbers
			response.ErrorNumber = 1
			response.ErrorMessage = err.Error()
		} else {
			response.Value = value
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
}

// handleAPI wraps an API handler function and returns an http.Handler.
// The handler function should return a value and an error.
// If the error is not nil, it will be returned as an Alpaca error response.
// If the error is nil, the value will be returned as an Alpaca response.
func handleAPI(handler func(r *http.Request) (any, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = addParamsToRequestContext(r)

		txID, err := getUintParam(r, "ClientTransactionID", true)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		response := baseResponse{
			ServerTransactionID: int(txCounter.Add(1)),
			ClientTransactionID: int(txID),
		}

		value, err := handler(r)
		if errors.Is(err, errBadRequest) {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		} else if errors.Is(err, errInternal) {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		} else if err != nil {
			// TODO: Define error numbers
			response.ErrorNumber = 1
			response.ErrorMessage = err.Error()
		} else {
			response.Value = value
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
}

// addParamsToRequestContext extracts the parameters from the request and adds
// them to the request context.
// PUT requests have the parameters in the body.
// GET requests have the parameters in the URL.
func addParamsToRequestContext(r *http.Request) *http.Request {
	var params url.Values

	if r.Method == "PUT" {
		params, _ = parseBodyParams(r)

	} else {
		params = r.URL.Query()
	}

	// Insert the params into the request context
	ctx := context.WithValue(r.Context(), paramsKey, params)

	return r.WithContext(ctx)
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

// getParam now reads the field from the request body.
func getParam(r *http.Request, field string, anyCase bool) (string, error) {
	params, ok := r.Context().Value(paramsKey).(url.Values)
	if !ok {
		return "", fmt.Errorf("%w: missing params", errBadRequest)
	}

	if !anyCase {
		param, ok := params[field]
		if ok {
			return param[0], nil
		}
		return "", fmt.Errorf("%w: missing field %s", errBadRequest, field)
	}

	for param, value := range params {
		if anyCase && strings.EqualFold(param, field) {
			return value[0], nil
		}
	}
	return "", fmt.Errorf("%w: missing field %s", errBadRequest, field)
}

func getBoolParam(r *http.Request, field string) (bool, error) {
	value, err := getParam(r, field, false)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(value)
}

func getFloatParam(r *http.Request, field string) (float64, error) {
	value, err := getParam(r, field, false)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(value, 64)
}

func getIntParam(r *http.Request, field string) (int, error) {
	value, err := getParam(r, field, false)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(value)
}

func getUintParam(r *http.Request, field string, anyCase bool) (uint, error) {
	value, err := getParam(r, field, anyCase)
	if err != nil {
		return 0, err
	}
	i, err := strconv.ParseUint(value, 10, 32)
	return uint(i), err
}
