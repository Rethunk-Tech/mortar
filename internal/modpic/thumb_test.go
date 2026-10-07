package modpic

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestThumbnailShrinksLargePicturesAndKeepsSmallOnes(t *testing.T) {
	t.Parallel()
	encode := func(w, h int, asJPEG bool) []byte {
		var buf bytes.Buffer
		img := image.NewNRGBA(image.Rect(0, 0, w, h))
		if asJPEG {
			_ = jpeg.Encode(&buf, img, nil)
		} else {
			_ = png.Encode(&buf, img)
		}
		return buf.Bytes()
	}
	for _, tc := range []struct {
		name         string
		src          []byte
		typ, wantTyp string
		w, h         int
	}{
		{"wide jpeg", encode(1920, 1080, true), "image/jpeg", "image/jpeg", thumbPx, thumbPx * 1080 / 1920},
		{"tall png", encode(500, 2000, false), "image/png", "image/png", thumbPx * 500 / 2000, thumbPx},
	} {
		out, typ, err := thumbnail(tc.src, tc.typ)
		if err != nil || typ != tc.wantTyp {
			t.Fatalf("%s: type %q err %v", tc.name, typ, err)
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
		if err != nil || cfg.Width != tc.w || cfg.Height != tc.h {
			t.Fatalf("%s: got %dx%d (%v), want %dx%d", tc.name, cfg.Width, cfg.Height, err, tc.w, tc.h)
		}
	}
	small := encode(96, 96, false)
	if out, _, err := thumbnail(small, "image/png"); err != nil || !bytes.Equal(out, small) {
		t.Fatalf("a picture under the limit must pass through unchanged: %v", err)
	}
}
