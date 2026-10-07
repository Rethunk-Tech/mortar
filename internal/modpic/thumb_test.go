package modpic

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"strings"
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
		{"wide jpeg", encode(1920, 1080, true), "image/jpeg", "image/jpeg", Thumb, Thumb * 1080 / 1920},
		{"tall png", encode(500, 2000, false), "image/png", "image/png", Thumb * 500 / 2000, Thumb},
	} {
		out, typ, err := Shrink(tc.src, tc.typ, Thumb)
		if err != nil || typ != tc.wantTyp {
			t.Fatalf("%s: type %q err %v", tc.name, typ, err)
		}
		cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
		if err != nil || cfg.Width != tc.w || cfg.Height != tc.h {
			t.Fatalf("%s: got %dx%d (%v), want %dx%d", tc.name, cfg.Width, cfg.Height, err, tc.w, tc.h)
		}
	}
	small := encode(96, 96, false)
	if out, _, err := Shrink(small, "image/png", Thumb); err != nil || !bytes.Equal(out, small) {
		t.Fatalf("a picture under the limit must pass through unchanged: %v", err)
	}
}

func TestSizeOfPicksHeroOnlyWhenAsked(t *testing.T) {
	t.Parallel()
	for px, want := range map[int]int{0: Thumb, Thumb: Thumb, 640: Thumb, -1: Thumb, Hero: Hero} {
		if got := sizeOf(px); got != want {
			t.Errorf("sizeOf(%d) = %d, want %d", px, got, want)
		}
	}
	if !strings.HasSuffix(SizedURL("https://gcdn.thunderstore.io/a.png", Hero), "&s=1920") {
		t.Error("a hero URL must name its size")
	}
}
