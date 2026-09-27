package httphandler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/sfedu-crm/internal/domain"
	"github.com/sfedu-crm/internal/repository"
	"github.com/sfedu-crm/internal/service"
)

const (
	defaultTrainerPhotoDir = "data/uploads/trainers"
	maxTrainerPhotoBytes   = 96 << 10  // 96 KiB hard limit after browser-side compression
	maxTrainerUploadBody   = 192 << 10 // multipart overhead included
	trainerPhotoURLPrefix  = "/uploads/trainers/"
)

type TrainerHandler struct {
	trainerSvc *service.TrainerService
	photoDir   string
}

func NewTrainerHandler(trainerSvc *service.TrainerService) *TrainerHandler {
	return &TrainerHandler{trainerSvc: trainerSvc, photoDir: defaultTrainerPhotoDir}
}

func (h *TrainerHandler) ListTrainers(w http.ResponseWriter, r *http.Request) {
	active := true
	filter := repository.TrainerFilter{IsActive: &active}
	if value := r.URL.Query().Get("specialization"); value != "" {
		spec := domain.TrainerSpecialization(value)
		filter.Specialization = &spec
	}
	filter.Search = r.URL.Query().Get("search")
	trainers, err := h.trainerSvc.List(r.Context(), filter)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, trainers)
}

func (h *TrainerHandler) ListAllTrainers(w http.ResponseWriter, r *http.Request) {
	trainers, err := h.trainerSvc.List(r.Context(), repository.TrainerFilter{})
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusOK, trainers)
}

func (h *TrainerHandler) GetTrainer(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid trainer id")
		return
	}
	trainer, err := h.trainerSvc.GetByID(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	if !trainer.IsActive {
		handleError(w, domain.ErrNotFound)
		return
	}
	respond(w, http.StatusOK, trainer)
}

func (h *TrainerHandler) CreateTrainer(w http.ResponseWriter, r *http.Request) {
	var input domain.CreateTrainerInput
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	trainer, err := h.trainerSvc.Create(r.Context(), input)
	if err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusCreated, trainer)
}

func (h *TrainerHandler) UpdateTrainer(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid trainer id")
		return
	}
	current, err := h.trainerSvc.GetByID(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	var input domain.UpdateTrainerInput
	if err := decode(w, r, &input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	trainer, err := h.trainerSvc.Update(r.Context(), id, input)
	if err != nil {
		handleError(w, err)
		return
	}
	if current.PhotoURL != trainer.PhotoURL {
		h.removeManagedPhoto(current.PhotoURL, "")
	}
	respond(w, http.StatusOK, trainer)
}

func (h *TrainerHandler) DeleteTrainer(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid trainer id")
		return
	}
	if err := h.trainerSvc.Delete(r.Context(), id); err != nil {
		handleError(w, err)
		return
	}
	respond(w, http.StatusNoContent, nil)
}

// UploadTrainerPhoto accepts only an already-compressed WebP/JPEG image. The browser
// performs resize/compression first, keeping bandwidth, database and disk usage low.
func (h *TrainerHandler) UploadTrainerPhoto(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid trainer id")
		return
	}
	current, err := h.trainerSvc.GetByID(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxTrainerUploadBody)
	if err := r.ParseMultipartForm(maxTrainerUploadBody); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			respondError(w, http.StatusRequestEntityTooLarge, "trainer photo is too large")
			return
		}
		respondError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	file, _, err := r.FormFile("photo")
	if err != nil {
		respondError(w, http.StatusBadRequest, "photo is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxTrainerPhotoBytes+1))
	if err != nil {
		respondError(w, http.StatusBadRequest, "could not read photo")
		return
	}
	if len(data) == 0 {
		respondError(w, http.StatusBadRequest, "photo is empty")
		return
	}
	if len(data) > maxTrainerPhotoBytes {
		respondError(w, http.StatusRequestEntityTooLarge, "trainer photo must be 96 KiB or smaller")
		return
	}
	ext, err := trainerPhotoExtension(data)
	if err != nil {
		respondError(w, http.StatusBadRequest, "only compressed WebP or JPEG photos are allowed")
		return
	}

	if err := os.MkdirAll(h.photoDir, 0o755); err != nil {
		respondError(w, http.StatusInternalServerError, "could not prepare photo storage")
		return
	}
	filename, err := randomTrainerPhotoFilename(ext)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not create photo name")
		return
	}
	fullPath := filepath.Join(h.photoDir, filename)
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		respondError(w, http.StatusInternalServerError, "could not save photo")
		return
	}

	photoURL := trainerPhotoURLPrefix + filename
	updated, err := h.trainerSvc.Update(r.Context(), id, domain.UpdateTrainerInput{
		Specialization:  current.Specialization,
		Bio:             current.Bio,
		PhotoURL:        photoURL,
		ExperienceYears: current.ExperienceYears,
		IsActive:        current.IsActive,
	})
	if err != nil {
		_ = os.Remove(fullPath)
		handleError(w, err)
		return
	}
	h.removeManagedPhoto(current.PhotoURL, filename)
	respond(w, http.StatusOK, updated)
}

// DeleteTrainerPhoto removes the trainer image from both the profile and local disk.
func (h *TrainerHandler) DeleteTrainerPhoto(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDFromPath(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid trainer id")
		return
	}
	current, err := h.trainerSvc.GetByID(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	updated, err := h.trainerSvc.Update(r.Context(), id, domain.UpdateTrainerInput{
		Specialization:  current.Specialization,
		Bio:             current.Bio,
		PhotoURL:        "",
		ExperienceYears: current.ExperienceYears,
		IsActive:        current.IsActive,
	})
	if err != nil {
		handleError(w, err)
		return
	}
	h.removeManagedPhoto(current.PhotoURL, "")
	respond(w, http.StatusOK, updated)
}

// ServeTrainerPhoto serves only generated trainer image names. Directory listing and
// traversal are unavailable, while immutable caching avoids repeat traffic.
func (h *TrainerHandler) ServeTrainerPhoto(w http.ResponseWriter, r *http.Request) {
	filename := r.PathValue("filename")
	if !validTrainerPhotoFilename(filename) {
		http.NotFound(w, r)
		return
	}
	fullPath := filepath.Join(h.photoDir, filename)
	if _, err := os.Stat(fullPath); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	if strings.HasSuffix(strings.ToLower(filename), ".webp") {
		w.Header().Set("Content-Type", "image/webp")
	} else {
		w.Header().Set("Content-Type", "image/jpeg")
	}
	http.ServeFile(w, r, fullPath)
}

func trainerPhotoExtension(data []byte) (string, error) {
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return ".webp", nil
	}
	if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
		return ".jpg", nil
	}
	return "", errors.New("unsupported trainer photo format")
}

func randomTrainerPhotoFilename(ext string) (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]) + ext, nil
}

func validTrainerPhotoFilename(filename string) bool {
	if filename == "" || filepath.Base(filename) != filename || strings.Contains(filename, "..") {
		return false
	}
	lower := strings.ToLower(filename)
	if !strings.HasSuffix(lower, ".webp") && !strings.HasSuffix(lower, ".jpg") {
		return false
	}
	dot := strings.LastIndexByte(filename, '.')
	if dot != 24 {
		return false
	}
	for _, c := range filename[:dot] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func (h *TrainerHandler) removeManagedPhoto(photoURL, keepFilename string) {
	if !strings.HasPrefix(photoURL, trainerPhotoURLPrefix) {
		return
	}
	filename := strings.TrimPrefix(photoURL, trainerPhotoURLPrefix)
	if filename == keepFilename || !validTrainerPhotoFilename(filename) {
		return
	}
	_ = os.Remove(filepath.Join(h.photoDir, filename))
}
