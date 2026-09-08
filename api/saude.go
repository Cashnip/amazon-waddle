package api

import "net/http"

// saude é a prova de que o conjunto sobe: sem banco, sem Redis, sem domínio.
func saude(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write([]byte(`{"status":"ok"}` + "\n"))
}
