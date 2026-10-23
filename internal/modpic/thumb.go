package modpic

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/gif" // registers the GIF decoder for image.Decode
	"image/jpeg"
	"image/png"
	"sync"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // registers the WebP decoder for image.Decode
)

// Thumb and Hero are the longest sides a cached picture keeps. Every tile and card shows a picture at most 72 CSS
// pixels wide (144 on a 2x screen), and the browser decodes a picture it shows at full size (a 5334x3122 mod banner
// is 66 MiB of pixels), so the cache stores a Thumb copy; only a profile's hero banner, which spans the window, asks
// for Hero.
const (
	Thumb = 192
	Hero  = 1920
)

// maxSourcePixels refuses to decode a picture that would need more than about 400 MiB of pixels.
const maxSourcePixels = 100_000_000

// decodeMu runs one downscale at a time, so a burst of large pictures holds one decoded original, not one per worker.
var decodeMu sync.Mutex

// animated reports a GIF with more than one frame, an animated PNG (an acTL chunk before its image data) or an
// animated WebP (an ANMF frame chunk).
func animated(b []byte, typ string) bool {
	switch typ {
	case "image/gif":
		// Each frame of an animation carries a graphic control extension.
		return bytes.Count(b, []byte{0x21, 0xF9, 0x04}) > 1
	case "image/png":
		for at := 8; at+8 <= len(b); {
			size := int(binary.BigEndian.Uint32(b[at:]))
			switch string(b[at+4 : at+8]) {
			case "acTL":
				return true
			case "IDAT":
				return false
			}
			at += 12 + size
		}
		return false
	case "image/webp":
		return bytes.Contains(b, []byte("ANMF"))
	}
	return false
}

// Shrink shrinks b to px on its longest side. A picture already that small is returned as it is; a
// downscaled JPEG stays a JPEG and anything else becomes a PNG, which keeps transparency.
func Shrink(b []byte, typ string, px int) ([]byte, string, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return nil, "", err
	}
	if cfg.Width*cfg.Height > maxSourcePixels {
		return nil, "", fmt.Errorf("the picture is %dx%d pixels, too large to shrink", cfg.Width, cfg.Height)
	}
	// A downscale keeps only the first frame, so an animated picture is kept whole; MaxSize already caps its bytes.
	if cfg.Width <= px && cfg.Height <= px || animated(b, typ) {
		return b, typ, nil
	}
	decodeMu.Lock()
	defer decodeMu.Unlock()
	src, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, "", err
	}
	w, h := px, px
	if cfg.Width > cfg.Height {
		h = max(1, cfg.Height*px/cfg.Width)
	} else {
		w = max(1, cfg.Width*px/cfg.Height)
	}
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	scaleBanded(dst, src)
	var out bytes.Buffer
	if typ == "image/jpeg" {
		err = jpeg.Encode(&out, dst, &jpeg.Options{Quality: 85})
		return out.Bytes(), typ, err
	}
	err = png.Encode(&out, dst)
	return out.Bytes(), "image/png", err
}

// scaleTmpBytes bounds the float scratch a scale passes through at once. A one-shot CatmullRom scale holds 32 bytes for
// every pixel of the destination width times the source height: 190 MiB for a 5334x3122 banner shrunk to Hero.
const scaleTmpBytes = 4 << 20

// scaleBanded is CatmullRom scaling of src into dst in two exact passes, each cut into bands that fit scaleTmpBytes:
// the horizontal pass by rows of src into a dst-wide copy, then the vertical pass by columns of that copy into dst.
// Bands of one size share a scaler, so its scratch is allocated once.
func scaleBanded(dst *image.NRGBA, src image.Image) {
	sr, w, h := src.Bounds(), dst.Bounds().Dx(), dst.Bounds().Dy()
	mid := image.NewNRGBA(image.Rect(0, 0, w, sr.Dy()))
	var sc xdraw.Scaler
	scaleN := 0
	rows := max(1, scaleTmpBytes/(w*32))
	for y := 0; y < sr.Dy(); y += rows {
		if n := min(rows, sr.Dy()-y); n != scaleN {
			sc, scaleN = xdraw.CatmullRom.NewScaler(w, n, sr.Dx(), n), n
		}
		sc.Scale(mid, image.Rect(0, y, w, y+scaleN), src, image.Rect(sr.Min.X, sr.Min.Y+y, sr.Max.X, sr.Min.Y+y+scaleN), xdraw.Src, nil)
	}
	sc, scaleN = nil, 0
	cols := max(1, scaleTmpBytes/(sr.Dy()*32))
	for x := 0; x < w; x += cols {
		if n := min(cols, w-x); n != scaleN {
			sc, scaleN = xdraw.CatmullRom.NewScaler(n, h, n, sr.Dy()), n
		}
		sc.Scale(dst, image.Rect(x, 0, x+scaleN, h), mid, image.Rect(x, 0, x+scaleN, sr.Dy()), xdraw.Src, nil)
	}
}
