package routes

import (
	"net/http"

	"pop/internal/handlers"
)

func RegisterPopRoutes(mux *http.ServeMux, h *handlers.PopHandler) {

	// 🔥 Root health check
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Application Up and Running"))
	})

	mux.HandleFunc("/pop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.Create(w, r)
			return
		}
		if r.Method == http.MethodGet {
			h.List(w, r)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/pop/stats", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.Stats(w, r)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/pop/trend", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.Trend(w, r)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/pop/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			h.Search(w, r)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
}
