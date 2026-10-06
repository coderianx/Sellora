package auth

import (
	"Sellora-Backend/internal/httpx"
	"Sellora-Backend/internal/models"
	"Sellora-Backend/internal/store"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

func RegisterHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()

	var req models.RegisterRequest

	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		httpx.SendJSON(
			w, http.StatusBadRequest,
			map[string]any{
				"error": "Invalid JSON body",
			},
		)
		return
	}

	if req.Username == "" {
		httpx.SendJSON(
			w, http.StatusBadRequest,
			map[string]any{
				"error": "username is required",
			},
		)

		return
	}

	isEmail := isValidEmail(req.Email)

	if !isEmail {
		httpx.SendJSON(
			w, http.StatusBadRequest,
			map[string]any{
				"error": "email is not a real email format",
			},
		)
		return
	}

	// hash password
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.Password),
		12,
	)

	if err != nil {
		httpx.SendJSON(
			w, http.StatusInternalServerError,
			map[string]any{
				"error": "Hash password error",
			},
		)
		return
	}

	// save to database
	_, err = store.DB.Exec(
		ctx,
		`
		INSERT INTO users (
			username, name, email, password
		)
		VALUES ($1,$2,$3,$4);
		`,
		req.Username,
		req.Name,
		req.Email,
		string(hashedPassword),
	)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			httpx.SendJSON(
				w, http.StatusConflict,
				map[string]any{
					"error": "user already exists",
				},
			)
			return
		}

		httpx.SendJSON(
			w, http.StatusInternalServerError,
			map[string]any{
				"error": "Database error",
			},
		)

		return
	}

	httpx.SendJSON(
		w, http.StatusCreated,
		map[string]any{
			"message": "user created successfully",
		},
	)
}

func LoginHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()

	var req models.LoginRequest

	err := json.NewDecoder(r.Body).Decode(&req)

	if err != nil {
		httpx.SendJSON(
			w, http.StatusBadRequest,
			map[string]any{
				"error": "Invalid JSON body",
			},
		)
		return
	}

	if req.Email == "" {
		httpx.SendJSON(
			w, http.StatusBadRequest,
			map[string]any{
				"error": "email is required",
			},
		)

		return
	}

	isEmail := isValidEmail(req.Email)

	if !isEmail {
		httpx.SendJSON(
			w, http.StatusBadRequest,
			map[string]any{
				"error": "email is not a real email format",
			},
		)
		return
	}

	if req.Password == "" {
		httpx.SendJSON(
			w, http.StatusBadRequest,
			map[string]any{
				"error": "password is required",
			},
		)
		return
	}

	// find user
	var hashedPassword string
	var useridDB int64
	var usernameDB string

	err = store.DB.QueryRow(
		ctx,
		`
		SELECT id, username, password
		FROM users
		WHERE email = $1;
		`,
		req.Email,
	).Scan(
		&useridDB,
		&usernameDB,
		&hashedPassword,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.SendJSON(
				w, http.StatusUnauthorized,
				map[string]any{
					"error": "invalid email or password",
				},
			)
			return
		}

		httpx.SendJSON(
			w, http.StatusInternalServerError,
			map[string]any{
				"error": "Database error",
			},
		)

		return
	}

	// compare password
	err = bcrypt.CompareHashAndPassword(
		[]byte(hashedPassword),
		[]byte(req.Password),
	)

	if err != nil {
		httpx.SendJSON(
			w, http.StatusUnauthorized,
			map[string]any{
				"error": "invalid email or password",
			},
		)
		return
	}

	// create jwt token
	claims := models.Claims{
		UserID:   useridDB,
		Username: usernameDB,

		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(
				time.Now(),
			),
			ExpiresAt: jwt.NewNumericDate(
				time.Now().Add(15 * time.Minute),
			),
		},
	}

	access := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	secret_key := os.Getenv("JWT_SECRET")

	if len(secret_key) == 0 {
		httpx.SendJSON(
			w, http.StatusInternalServerError,
			map[string]any{
				"error": "JWT Secret is missing",
			},
		)

		fmt.Println("[HARD ERROR] JWT_SECRET is not set in .env file")
		os.Exit(1)
	}

	accessToken, err := access.SignedString(
		secret_key,
	)

	if err != nil {
		httpx.SendJSON(
			w, http.StatusInternalServerError,
			map[string]any{
				"error": "Create JWT error",
			},
		)
		return
	}

	// create refresh token
	randomToken, err := generateRefreshToken(64)

	if err != nil {
		httpx.SendJSON(
			w, http.StatusInternalServerError,
			map[string]any{
				"error": "Create refresh token error",
			},
		)
		return
	}

	// save refresh token to database
	refreshToken := sha256.Sum256([]byte(randomToken))

	_, err = store.DB.Exec(
		ctx,
		`
		INSERT INTO refresh_tokens (user_id, token, expire_at)
		VALUES ($1,$2,$3)
		`,
		useridDB,
		hex.EncodeToString(refreshToken[:]),
		time.Now().Add(30*24*time.Hour),
	)

	if err != nil {
		httpx.SendJSON(
			w, http.StatusInternalServerError,
			map[string]any{
				"error": "Database error",
			},
		)
		return
	}

	httpx.SendJSON(
		w, http.StatusOK,
		map[string]any{
			"message":      "login successful",
			"accessToken":  accessToken,
			"refreshToken": randomToken,
		},
	)
}
