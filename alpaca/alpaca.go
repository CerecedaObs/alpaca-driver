package alpaca

import (
	"encoding/json"
	"errors"
	"net/http"
)

var ErrInvalidContentType = errors.New("invalid content type")

func handleResponse(w http.ResponseWriter, value any) {
	response := baseResponse{
		ServerTransactionID: int(txCounter.Add(1)),
	}
	if value != nil {
		response.Value = value
	}
	json.NewEncoder(w).Encode(response)
}

func handleError(w http.ResponseWriter, code int, message string) {
	response := baseResponse{
		ServerTransactionID: int(txCounter.Add(1)),
		ErrorNumber:         code,
		ErrorMessage:        message,
	}
	json.NewEncoder(w).Encode(response)
}

func parseRequest(r *http.Request, v any) error {
	if r.Header.Get("Content-Type") != "application/json" {
		return ErrInvalidContentType
	}

	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(v)
}
