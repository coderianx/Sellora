package store

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func ConnectPostreSQL() {
	var err error

	DB_URL := os.Getenv("DB_URL")

	if len(DB_URL) == 0 {
		fmt.Println("[ERROR] DB_URL is not set in .env file")
		os.Exit(1)
	}

	DB, err = pgxpool.New(
		context.Background(),
		DB_URL,
	)

	if err != nil {
		fmt.Println("[ERROR] Connecting database error: ", err)
		os.Exit(1)
	}

	fmt.Println("[INFO] Database connected")

}

func CreateTables() {
	_, err := DB.Exec(
		context.Background(),
		`
		CREATE TABLE IF NOT EXISTS users (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
		`,
	)

	if err != nil {
		fmt.Println("[ERROR] Creating 'users' table error: ", err)
		os.Exit(1)
	}

	_, err = DB.Exec(
		context.Background(),
		`
		CREATE TABLE IF NOT EXISTS refresh_tokens (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token TEXT NOT NULL,
			expire_at TIMESTAMP NOT NULL
		);
		`,
	)

	if err != nil {
		fmt.Println("[ERROR] Creating 'refresh_tokens' table error: ", err)
		os.Exit(1)
	}

	_, err = DB.Exec(
		context.Background(),
		`
		CREATE TABLE IF NOT EXISTS products (
			id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL,
			price NUMERIC NOT NULL,
			stock INTEGER NOT NULL DEFAULT 0,
			categorys TEXT[] NOT NULL,
			object_keys TEXT[] NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);
		`,
	)

	if err != nil {
		fmt.Println("[ERROR] Creating 'products' table error: ", err)
		os.Exit(1)
	}

	_, err = DB.Exec(
		context.Background(),
		`
		ALTER TABLE products
		ADD COLUMN IF NOT EXISTS stock INTEGER NOT NULL DEFAULT 0;
		`,
	)

	if err != nil {
		fmt.Println("[ERROR] Adding 'products.stock' column error: ", err)
		os.Exit(1)
	}

	fmt.Println("[INFO] Tables created")
}
