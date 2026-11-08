package cp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"slices"
	"strconv"
	"strings"
)

func ConflictEvidence(kind string, hits []packHit) []Evidence {
	out := []Evidence{}
	var bounds [][]patchBounds
	for _, hit := range hits {
		patches, clashes := hit.Edits, hit.Clashes
		if kind == "load" {
			patches, clashes = hit.Loads, hit.LoadClashes
		}
		for i, patch := range patches {
			if clashes != nil && !clashes[i] {
				continue
			}
			e := Evidence{
				PackName: hit.Name,
				PackID:   hit.ID,
				Source:   patch.Source,
				Index:    patch.Index,
				Action:   patch.Action,
				Target:   patch.Target,
				ToArea:   patch.ToArea,
				FromArea: patch.ImageFromArea,
				When:     patch.When,
				Priority: patch.Priority,
				FromFile: patch.FromFile,
				Keys:     sharedKeys(patch, hits, hit.ID),
			}
			if patch.Image {
				if bounds == nil {
					bounds = editBounds(hits)
				}
				if ox, oy, ow, oh, ok := overlapAgainst(patch, hits, bounds, hit.ID); ok {
					e.CropX, e.CropY, e.CropW, e.CropH = sourceCrop(patch, ox, oy, ow, oh)
				}
			}
			out = append(out, e)
		}
	}
	return out
}

// sharedKeys are the data entries and fields of patch that another pack's edit also writes, as the pack scoped
// token keys the shapes hold.
func sharedKeys(patch Patch, hits []packHit, self ModID) []string {
	keys := []string{}
	for _, s := range patch.Shapes {
		if s.Kind != 'p' {
			continue
		}
		clash := slices.ContainsFunc(hits, func(h packHit) bool {
			return !Equal(h.ID, self) && slices.ContainsFunc(h.Edits, func(o Patch) bool {
				return slices.ContainsFunc(o.Shapes, s.overlaps)
			})
		})
		if clash {
			keys = append(keys, s.Key)
		}
	}
	return keys
}

// patchBounds is one edit's destination rectangle, computed once per conflict rather than once per
// pair of clashing patches.
type patchBounds struct {
	R  image.Rectangle
	Ok bool
}

func editBounds(hits []packHit) [][]patchBounds {
	out := make([][]patchBounds, len(hits))
	for i, hit := range hits {
		out[i] = make([]patchBounds, len(hit.Edits))
		for j, edit := range hit.Edits {
			out[i][j].R, out[i][j].Ok = destBounds(edit)
		}
	}
	return out
}

func overlapAgainst(patch Patch, hits []packHit, bounds [][]patchBounds, self ModID) (x, y, w, h int, ok bool) {
	own, ok := destBounds(patch)
	if !ok {
		return 0, 0, 0, 0, false
	}
	var inter image.Rectangle
	found := false
	for i, hit := range hits {
		if Equal(hit.ID, self) {
			continue
		}
		for _, peer := range bounds[i] {
			if !peer.Ok {
				continue
			}
			r := own.Intersect(peer.R)
			if r.Empty() {
				continue
			}
			if !found {
				inter = r
				found = true
			} else {
				inter = inter.Union(r)
			}
		}
	}
	if !found {
		return 0, 0, 0, 0, false
	}
	return inter.Min.X, inter.Min.Y, inter.Dx(), inter.Dy(), true
}

func destBounds(p Patch) (image.Rectangle, bool) {
	var r image.Rectangle
	found := false
	add := func(x, y, w, h int) {
		if w <= 0 || h <= 0 {
			return
		}
		nr := image.Rect(x, y, x+w, y+h)
		if !found {
			r = nr
			found = true
			return
		}
		r = r.Union(nr)
	}
	for _, s := range p.Shapes {
		if s.Kind == 'p' || s.Kind == 'w' {
			continue
		}
		if s.Cells != "" {
			for cell := range strings.SplitSeq(s.Cells, ";") {
				xs, ys, isCell := strings.Cut(cell, ",")
				if !isCell {
					continue
				}
				cx, errX := strconv.Atoi(xs)
				cy, errY := strconv.Atoi(ys)
				if errX != nil || errY != nil {
					continue
				}
				add(cx*16, cy*16, 16, 16)
			}
			continue
		}
		x, y, w, h := s.area()
		add(x, y, w, h)
	}
	return r, found
}

func sourceCrop(p Patch, ox, oy, ow, oh int) (int, int, int, int) {
	fx, fy := 0, 0
	if from, ok := areaOf(json.RawMessage(p.ImageFromArea)); ok {
		fx, fy = from.X, from.Y
	}
	tx, ty := 0, 0
	if to, ok := destBounds(p); ok {
		tx, ty = to.Min.X, to.Min.Y
	}
	return fx + (ox - tx), fy + (oy - ty), ow, oh
}

func CropPackImage(root, fromFile string, x, y, w, h int) (string, error) {
	raw, ok := readPackPath(root, fromFile)
	if !ok {
		return "", errors.New("missing image")
	}
	return cropImageDataURL(raw, x, y, w, h)
}

func cropImageDataURL(raw []byte, x, y, w, h int) (string, error) {
	if !pngDecodable(raw) {
		return "", errors.New("image is missing, malformed or too large")
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	r := image.Rect(x, y, x+w, y+h).Intersect(img.Bounds())
	if r.Empty() {
		return "", errors.New("crop is empty")
	}
	out := image.NewNRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(out, out.Bounds(), img, r.Min, draw.Src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
