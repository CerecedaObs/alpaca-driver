package alpaca

import (
	"encoding/json"
	"net/http"
)

func handleResponse(w http.ResponseWriter, value interface{}) {
	response := baseResponse{
		ServerTransactionID: int(txCounter.Add(1)),
		Value:               value,
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
