package main

import (
	"Sellora-Backend/internal/handlers/auth"
	"Sellora-Backend/internal/handlers/product"
	"Sellora-Backend/internal/httpx"
	"Sellora-Backend/internal/middlewares"
	"Sellora-Backend/internal/store"
	"fmt"
	"net/http"
	"os"
	"time"

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
	// Connect Redis
	store.ConnectRedis()
	// Connect B2
	store.ConnectB2()

	// Chi router
	router := chi.NewRouter()

	router.Route("/api/v1", func(r chi.Router) {
		r.With(
			middlewares.RateLimitByIP(
				5,
				time.Minute,
			),
		).Get("/health", func(w http.ResponseWriter, r *http.Request) {
			httpx.SendJSON(w, 200, map[string]any{
				"message": "server ok",
			})
		})

		r.With(
			middlewares.RateLimitByIP(
				3,
				time.Minute,
			),
		).Post(
			"/auth/register", auth.RegisterHandler,
		)

		r.With(
			middlewares.RateLimitByIP(
				5,
				time.Minute,
			),
		).Post(
			"/auth/login", auth.LoginHandler,
		)

		r.With(
			middlewares.RateLimitByIP(
				5,
				time.Minute,
			),
			middlewares.RequireAuth,
		).Delete(
			"/auth/delete", auth.DeleteUser,
		)

		r.With(
			middlewares.RateLimitByIP(
				10,
				time.Minute,
			),
			middlewares.RequireAuth,
		).Post(
			"/products", product.CreateProductHandler,
		)

	})

	// Start server
	fmt.Println("[INFO] Server is starting...")
	http.ListenAndServe(":8080", router)
}
