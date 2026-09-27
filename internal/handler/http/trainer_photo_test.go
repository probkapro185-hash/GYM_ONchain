package httphandler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTrainerPhotoExtension(t *testing.T) {
	webp := []byte("RIFF\x00\x00\x00\x00WEBP")
	if ext, err := trainerPhotoExtension(webp); err != nil || ext != ".webp" {
		t.Fatalf("expected webp, got %q err=%v", ext, err)
	}
	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0}
	if ext, err := trainerPhotoExtension(jpeg); err != nil || ext != ".jpg" {
		t.Fatalf("expected jpg, got %q err=%v", ext, err)
	}
	if _, err := trainerPhotoExtension([]byte("not-an-image")); err == nil {
		t.Fatal("expected unsupported format error")
	}
}

func TestTrainerPhotoFilenameValidation(t *testing.T) {
	good := "0123456789abcdef01234567.webp"
	if !validTrainerPhotoFilename(good) {
		t.Fatalf("expected %q to be valid", good)
	}
	for _, bad := range []string{
		"../" + good,
		"0123456789abcdef0123456.webp",
		"0123456789abcdef0123456z.webp",
		"0123456789abcdef01234567.png",
		"",
	} {
		if validTrainerPhotoFilename(bad) {
			t.Fatalf("expected %q to be rejected", bad)
		}
	}
}

func TestRandomTrainerPhotoFilename(t *testing.T) {
	name, err := randomTrainerPhotoFilename(".webp")
	if err != nil {
		t.Fatal(err)
	}
	if !validTrainerPhotoFilename(name) {
		t.Fatalf("generated invalid filename: %q", name)
	}
}

func TestServeTrainerPhoto(t *testing.T) {
	dir := t.TempDir()
	name := "0123456789abcdef01234567.jpg"
	if err := os.WriteFile(filepath.Join(dir, name), []byte{0xff, 0xd8, 0xff, 0xe0}, 0o644); err != nil {
		t.Fatal(err)
	}
	h := &TrainerHandler{photoDir: dir}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /uploads/trainers/{filename}", h.ServeTrainerPhoto)

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/uploads/trainers/"+name, nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if got := rr.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
		t.Fatalf("expected immutable cache header, got %q", got)
	}
	if got := rr.Header().Get("Content-Type"); !strings.HasPrefix(got, "image/jpeg") {
		t.Fatalf("expected jpeg content type, got %q", got)
	}

	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/uploads/trainers/not-safe.jpg", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for invalid filename, got %d", rr.Code)
	}
}
