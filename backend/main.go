package backend

import (
	"Sellora-Backend/internal/store"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	err := godotenv.Load()

	if err != nil {
		fmt.Println("[ERROR] Load .env error: ", err)
		os.Exit(1)
	}

	// Connect databases
	store.ConnectPostreSQL()
	// Create Tables
	store.CreateTables()

	router := chi.NewRouter()

	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			w.Header().Set("Content-Type", "application/json")

			w.WriteHeader(http.StatusOK)

			json.NewEncoder(w).Encode(map[string]any{
				"message": "Server OK",
			})

		})
	})

	// Start server
	fmt.Println("[INFO] Server is starting...")
	http.ListenAndServe(":8080", router)
}
