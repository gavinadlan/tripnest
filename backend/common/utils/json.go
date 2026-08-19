package utils

import (
	"encoding/json"
	"log"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

// ReadJSON membaca body JSON ke dst.
func ReadJSON(r *http.Request, dst interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

// WriteJSON menulis response JSON dengan status code tertentu.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

// WriteError menulis response error.
//
// Untuk status 4xx pesan error dikirim apa adanya: itu pesan validasi yang
// memang kita tulis sendiri untuk dibaca pemanggil. Untuk status 5xx pesan
// aslinya hanya dicatat di log dan pemanggil menerima pesan generik --
// error Postgres mentah membocorkan struktur skema.
func WriteError(w http.ResponseWriter, status int, err error) {
	message := "unknown error"
	if err != nil {
		message = err.Error()
	}

	if status >= http.StatusInternalServerError {
		log.Printf("[%d] internal error: %v", status, err)
		message = "terjadi kesalahan pada server"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
