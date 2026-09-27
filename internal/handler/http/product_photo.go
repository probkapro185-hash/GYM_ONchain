package httphandler

import (
	"bytes"
	"errors"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sfedu-crm/internal/domain"
)

const maxProductPhotoBytes = 256 * 1024
const productPhotoPrefix = "/uploads/products/"

func validateProductPhoto(data []byte) error {
	if len(data) == 0 || len(data) > maxProductPhotoBytes {
		return errors.New("Фото должно быть не больше 256 КБ")
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return errors.New("Выберите корректное изображение JPEG")
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 1600 || cfg.Height > 1600 {
		return errors.New("Размеры фото не должны превышать 1600 пикселей")
	}
	if _, err := jpeg.Decode(bytes.NewReader(data)); err != nil {
		return errors.New("Файл JPEG повреждён")
	}
	return nil
}

func (h *ShopHandler) UploadProductPhoto(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, 400, "invalid product id")
		return
	}
	if _, err := h.shopSvc.GetProduct(r.Context(), domain.RoleAdmin, id); err != nil {
		handleError(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxProductPhotoBytes+32*1024)
	if err := r.ParseMultipartForm(maxProductPhotoBytes + 32*1024); err != nil {
		respondError(w, 400, "invalid or oversized multipart form")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	f, _, err := r.FormFile("photo")
	if err != nil {
		respondError(w, 400, "photo is required")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxProductPhotoBytes+1))
	if err != nil {
		respondError(w, 400, "could not read photo")
		return
	}
	if len(data) > maxProductPhotoBytes {
		respondError(w, 413, "Фото должно быть не больше 256 КБ")
		return
	}
	if err := validateProductPhoto(data); err != nil {
		respondError(w, 400, err.Error())
		return
	}
	if err := os.MkdirAll(h.photoDir, 0755); err != nil {
		respondError(w, 500, "could not prepare photo storage")
		return
	}
	filename, err := randomTrainerPhotoFilename(".jpg")
	if err != nil {
		respondError(w, 500, "could not create photo name")
		return
	}
	path := filepath.Join(h.photoDir, filename)
	if err := os.WriteFile(path, data, 0644); err != nil {
		respondError(w, 500, "could not save photo")
		return
	}
	product, oldURL, err := h.shopSvc.SetProductPhoto(r.Context(), id, productPhotoPrefix+filename)
	if err != nil {
		_ = os.Remove(path)
		handleError(w, err)
		return
	}
	h.removeProductPhoto(oldURL)
	respond(w, 200, product)
}
func (h *ShopHandler) DeleteProductPhoto(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, 400, "invalid product id")
		return
	}
	product, oldURL, err := h.shopSvc.SetProductPhoto(r.Context(), id, "")
	if err != nil {
		handleError(w, err)
		return
	}
	h.removeProductPhoto(oldURL)
	respond(w, 200, product)
}
func (h *ShopHandler) removeProductPhoto(url string) {
	if !strings.HasPrefix(url, productPhotoPrefix) {
		return
	}
	name := strings.TrimPrefix(url, productPhotoPrefix)
	if validTrainerPhotoFilename(name) && strings.HasSuffix(name, ".jpg") {
		_ = os.Remove(filepath.Join(h.photoDir, name))
	}
}
func (h *ShopHandler) ServeProductPhoto(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("filename")
	if !validTrainerPhotoFilename(name) || !strings.HasSuffix(name, ".jpg") {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(h.photoDir, name)
	if _, err := os.Stat(path); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeFile(w, r, path)
}
