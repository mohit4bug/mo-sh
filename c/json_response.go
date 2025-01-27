package c

import (
	"encoding/json"
	"net/http"
)

type JSON map[string]interface{}

func JSONResponse(w http.ResponseWriter, status int, body JSON) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
