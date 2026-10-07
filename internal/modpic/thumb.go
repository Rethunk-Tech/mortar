package modpic

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif" // registers the GIF decoder for image.Decode
	"image/jpeg"
	"image/png"
	"sync"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp" // registers the WebP decoder for image.Decode
)

// thumbPx is the longest side a cached picture keeps. The UI shows pictures at most 72 CSS pixels wide, and the
// browser decodes every picture it shows at full size (a 5334x3122 mod banner is 66 MiB of pixels), so the cache
// stores a thumbnail instead of the original.
const thumbPx = 192

// maxSourcePixels refuses to decode a picture that would need more than about 400 MiB of pixels.
const maxSourcePixels = 100_000_000

// decodeMu runs one downscale at a time, so a burst of large pictures holds one decoded original, not one per worker.
var decodeMu sync.Mutex

// thumbnail shrinks b to thumbPx on its longest side. A picture already that small is returned as it is; a
// downscaled JPEG stays a JPEG and anything else becomes a PNG, which keeps transparency.
func thumbnail(b []byte, typ string) ([]byte, string, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		return nil, "", err
	}
	if cfg.Width <= thumbPx && cfg.Height <= thumbPx {
		return b, typ, nil
	}
	if cfg.Width*cfg.Height > maxSourcePixels {
		return nil, "", fmt.Errorf("the picture is %dx%d pixels, too large to shrink", cfg.Width, cfg.Height)
	}
	decodeMu.Lock()
	defer decodeMu.Unlock()
	src, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, "", err
	}
	w, h := thumbPx, thumbPx
	if cfg.Width > cfg.Height {
		h = max(1, cfg.Height*thumbPx/cfg.Width)
	} else {
		w = max(1, cfg.Width*thumbPx/cfg.Height)
	}
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Src, nil)
	var out bytes.Buffer
	if typ == "image/jpeg" {
		err = jpeg.Encode(&out, dst, &jpeg.Options{Quality: 85})
		return out.Bytes(), typ, err
	}
	err = png.Encode(&out, dst)
	return out.Bytes(), "image/png", err
}
