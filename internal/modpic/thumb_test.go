package modpic

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"

	xdraw "golang.org/x/image/draw"
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

func TestAnAnimatedPictureKeepsEveryFrame(t *testing.T) {
	t.Parallel()
	frame := image.NewPaletted(image.Rect(0, 0, 400, 400), color.Palette{color.Black, color.White})
	var gifBuf bytes.Buffer
	if err := gif.EncodeAll(&gifBuf, &gif.GIF{Image: []*image.Paletted{frame, frame}, Delay: []int{10, 10}}); err != nil {
		t.Fatal(err)
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, image.NewNRGBA(image.Rect(0, 0, 400, 400))); err != nil {
		t.Fatal(err)
	}
	// An animated PNG is a PNG with an acTL chunk after its header; the IHDR chunk ends at byte 33.
	still := pngBuf.Bytes()
	actl := []byte{0, 0, 0, 8, 'a', 'c', 'T', 'L', 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0}
	apng := append(append(append([]byte{}, still[:33]...), actl...), still[33:]...)
	for name, tc := range map[string]struct {
		b   []byte
		typ string
	}{"gif": {gifBuf.Bytes(), "image/gif"}, "apng": {apng, "image/png"}} {
		out, typ, err := Shrink(tc.b, tc.typ, Thumb)
		if err != nil || typ != tc.typ || !bytes.Equal(out, tc.b) {
			t.Errorf("%s: an animated picture was re-encoded (%d -> %d bytes, %s, %v)", name, len(tc.b), len(out), typ, err)
		}
	}
	if out, _, err := Shrink(still, "image/png", Thumb); err != nil || bytes.Equal(out, still) {
		t.Errorf("a still picture was not shrunk: %v", err)
	}
}

func TestBandedScaleMatchesOneShotAcrossBandEdges(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 800, 1200))
	for y := range 1200 {
		for x := range 800 {
			i := src.PixOffset(x, y)
			src.Pix[i], src.Pix[i+1], src.Pix[i+2], src.Pix[i+3] = byte(x/7&0xff), byte(y/9&0xff), byte((x+y)/30&0xff), 255
		}
	}
	want := image.NewNRGBA(image.Rect(0, 0, 320, 480))
	xdraw.CatmullRom.Scale(want, want.Bounds(), src, src.Bounds(), xdraw.Src, nil)
	got := image.NewNRGBA(want.Bounds())
	scaleBanded(got, src)
	for i := range want.Pix {
		if d := int(want.Pix[i]) - int(got.Pix[i]); d > 2 || d < -2 {
			t.Fatalf("pixel byte %d differs by %d", i, d)
		}
	}
}
