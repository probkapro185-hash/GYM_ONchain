package httphandler

import (
	"bytes"
	"image"
	"image/jpeg"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestProductPhotoValidation(t *testing.T) {
	encode := func(w, h int) []byte {
		var b bytes.Buffer
		if err := jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, w, h)), nil); err != nil {
			t.Fatal(err)
		}
		return b.Bytes()
	}
	valid := encode(30, 20)
	if err := validateProductPhoto(valid); err != nil {
		t.Fatal(err)
	}
	for name, data := range map[string][]byte{"empty": nil, "fake": []byte("<svg onload='alert(1)'/>"), "truncated": valid[:len(valid)/2], "too-large": make([]byte, maxProductPhotoBytes+1), "too-wide": encode(1601, 1)} {
		t.Run(name, func(t *testing.T) {
			if validateProductPhoto(data) == nil {
				t.Fatal("invalid image accepted")
			}
		})
	}
}
func TestProductPhotoStoragePaths(t *testing.T) {
	dir := t.TempDir()
	handler := &ShopHandler{photoDir: dir}
	name := "aaaaaaaaaaaaaaaaaaaaaaaa.jpg"
	if err := os.WriteFile(filepath.Join(dir, name), []byte("sample"), 0600); err != nil {
		t.Fatal(err)
	}
	handler.removeProductPhoto("/uploads/trainers/" + name)
	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		t.Fatal("unmanaged URL removed photo")
	}
	for _, bad := range []string{"../secret.jpg", "a.jpg", "aaaaaaaaaaaaaaaaaaaaaaaa.svg"} {
		req := httptest.NewRequest("GET", "/uploads/products/ignored", nil)
		req.SetPathValue("filename", bad)
		rec := httptest.NewRecorder()
		handler.ServeProductPhoto(rec, req)
		if rec.Code != 404 {
			t.Fatalf("unsafe filename accepted: %q", bad)
		}
	}
	handler.removeProductPhoto(productPhotoPrefix + name)
	if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
		t.Fatal("managed photo was not removed")
	}
}
