package product

import (
	"Sellora-Backend/internal/httpx"
	"Sellora-Backend/internal/models"
	"Sellora-Backend/internal/store"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

func CreateProductHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx := r.Context()

	r.Body = http.MaxBytesReader(w, r.Body, 32<<20)

	err := r.ParseMultipartForm(32 << 20)

	if err != nil {
		var maxBytes *http.MaxBytesError

		if errors.As(err, &maxBytes) {
			httpx.SendJSON(
				w, http.StatusRequestEntityTooLarge,
				map[string]any{
					"error": "Files are too large",
				},
			)
			return
		}

		httpx.SendJSON(
			w, http.StatusBadRequest,
			map[string]any{
				"error": "Invalid multipart body",
			},
		)
		return
	}

	defer r.MultipartForm.RemoveAll()

	priceRaw := strings.TrimSpace(r.FormValue("price"))
	stockRaw := strings.TrimSpace(r.FormValue("stock"))

	req := models.CreateProductRequest{
		Name:        strings.TrimSpace(r.FormValue("name")),
		Description: r.FormValue("description"),
		Categorys:   parseCategorys(r.Form["categorys"]),
	}

	if req.Name == "" {
		httpx.SendJSON(
			w, http.StatusBadRequest,
			map[string]any{
				"error": "Name is required",
			},
		)
		return
	}

	if priceRaw == "" {
		httpx.SendJSON(
			w, http.StatusBadRequest,
			map[string]any{
				"error": "Price is required",
			},
		)
		return
	}

	req.Price, err = strconv.ParseFloat(priceRaw, 64)

	if err != nil || req.Price <= 0 {
		httpx.SendJSON(
			w, http.StatusBadRequest,
			map[string]any{
				"error": "Invalid price",
			},
		)
		return
	}

	if stockRaw != "" {
		req.Stock, err = strconv.Atoi(stockRaw)

		if err != nil || req.Stock < 0 {
			httpx.SendJSON(
				w, http.StatusBadRequest,
				map[string]any{
					"error": "Invalid stock",
				},
			)
			return
		}
	}

	files := r.MultipartForm.File["files"]

	if len(files) == 0 {
		httpx.SendJSON(
			w, http.StatusBadRequest,
			map[string]any{
				"error": "Files are required",
			},
		)
		return
	}

	if len(files) > 8 {
		httpx.SendJSON(
			w, http.StatusBadRequest,
			map[string]any{
				"error": "Too many files",
			},
		)
		return
	}

	if store.B2Client == nil {
		httpx.SendJSON(
			w, http.StatusInternalServerError,
			map[string]any{
				"error": "File storage is unavailable",
			},
		)
		return
	}

	objectKeys := make([]string, 0, len(files))

	for _, header := range files {
		file, err := header.Open()

		if err != nil {
			deleteUploaded(objectKeys)

			httpx.SendJSON(
				w, http.StatusBadRequest,
				map[string]any{
					"error": "Invalid file",
				},
			)
			return
		}

		objectKey, err := store.UploadFile(
			store.B2Client,
			file,
			header.Filename,
			header.Header.Get("Content-Type"),
		)

		file.Close()

		if err != nil {
			deleteUploaded(objectKeys)

			httpx.SendJSON(
				w, http.StatusInternalServerError,
				map[string]any{
					"error": "File upload error",
				},
			)
			return
		}

		objectKeys = append(objectKeys, objectKey)
	}

	var productID int64

	err = store.DB.QueryRow(
		ctx,
		`
		INSERT INTO products (
			name, description, price, stock, categorys, object_keys
		)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id;
		`,
		req.Name,
		req.Description,
		req.Price,
		req.Stock,
		req.Categorys,
		objectKeys,
	).Scan(&productID)

	if err != nil {
		deleteUploaded(objectKeys)

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
			"message":     "product created successfully",
			"id":          productID,
			"object_keys": objectKeys,
		},
	)
}

func parseCategorys(values []string) []string {
	if len(values) == 1 && strings.Contains(values[0], ",") {
		values = strings.Split(values[0], ",")
	}

	categorys := make([]string, 0, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)

		if value != "" {
			categorys = append(categorys, value)
		}
	}

	return categorys
}

func deleteUploaded(keys []string) {
	if store.B2Client == nil {
		return
	}

	for _, key := range keys {
		_ = store.DeleteFile(store.B2Client, key)
	}
}
