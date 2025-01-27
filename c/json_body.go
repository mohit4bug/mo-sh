package c

import (
	"encoding/json"
	"net/http"
)

func JSONParseRequestBody(w http.ResponseWriter, r *http.Request, v any) {
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(v)
	if err != nil {
		JSONResponse(w, http.StatusBadRequest, JSON{
			"message": "Invalid request body",
		})
	}
}
