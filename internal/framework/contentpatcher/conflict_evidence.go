package contentpatcher

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/framework"
	"github.com/Rethunk-Tech/mortar/internal/mod"
)

func conflictEvidence(kind string, hits []packHit) []framework.ConflictEvidence {
	out := []framework.ConflictEvidence{}
	var bounds [][]patchBounds
	for _, hit := range hits {
		patches, clashes := hit.edits, hit.clashes
		if kind == "load" {
			patches, clashes = hit.loads, hit.loadClashes
		}
		for i, patch := range patches {
			if clashes != nil && !clashes[i] {
				continue
			}
			e := framework.ConflictEvidence{
				PackName: hit.name,
				PackID:   hit.id,
				Source:   patch.source,
				Index:    patch.index,
				Action:   patch.action,
				Target:   patch.target,
				ToArea:   patch.toArea,
				FromArea: patch.imageFromArea,
				When:     whenSummary(patch.when),
				Priority: patch.priority,
				FromFile: patch.fromFile,
				Keys:     clashingKeys(patch, hits, hit.id),
			}
			if patch.image {
				if bounds == nil {
					bounds = editBounds(hits)
				}
				if ox, oy, ow, oh, ok := overlapAgainst(patch, hits, bounds, hit.id); ok {
					e.CropX, e.CropY, e.CropW, e.CropH = sourceCrop(patch, ox, oy, ow, oh)
				}
			}
			out = append(out, e)
		}
	}
	return out
}

// clashingKeys lists the data entries and fields of patch that another pack's edit also writes, as
// "Entry 301" or "Field 301.Price" with any TargetField path, without the pack scoping of token keys.
func clashingKeys(patch cpPatch, hits []packHit, self mod.ID) []string {
	keys := []string{}
	for _, s := range patch.shapes {
		if s.kind != 'p' {
			continue
		}
		clash := slices.ContainsFunc(hits, func(h packHit) bool {
			return !mod.Equal(h.id, self) && slices.ContainsFunc(h.edits, func(o cpPatch) bool {
				return slices.ContainsFunc(o.shapes, s.overlaps)
			})
		})
		if clash {
			keys = append(keys, keyLabel(s.key))
		}
	}
	return keys
}

func keyLabel(key string) string {
	label := "Entry "
	if rest, ok := strings.CutPrefix(key, "field:"); ok {
		label, key = "Field ", rest
	} else {
		key = strings.TrimPrefix(key, "entry:")
	}
	return label + packScope.ReplaceAllString(key, "")
}

// packScope matches the "@<pack folder>|" packScopedKey puts before a token key.
var packScope = regexp.MustCompile(`@[^|]*\|`)

func whenSummary(w cpWhen) string {
	var parts []string
	for _, group := range w.anyOf {
		parts = append(parts, "HasMod "+strings.Join(group, ", "))
	}
	for _, id := range w.noneOf {
		parts = append(parts, "not HasMod "+id)
	}
	for _, c := range w.config {
		parts = append(parts, c.field+"="+strings.Join(c.values, "|"))
	}
	for _, d := range w.dynamic {
		label := d.name
		if d.contains != "" {
			label += " contains " + d.contains
		} else if len(d.values) > 0 {
			label += "=" + strings.Join(d.values, ", ")
		}
		if !d.expected {
			label = "not " + label
		}
		parts = append(parts, label)
	}
	for _, f := range w.flags {
		if f.present {
			parts = append(parts, "HasFlag "+f.name)
		} else {
			parts = append(parts, "not HasFlag "+f.name)
		}
	}
	if w.spouse != "" && w.spouse != "\x00" {
		parts = append(parts, "Spouse "+w.spouse)
	}
	for _, token := range slices.Sorted(maps.Keys(w.places)) {
		parts = append(parts, token+"="+strings.Join(w.places[token], ", "))
	}
	return strings.Join(parts, "; ")
}

// patchBounds is one edit's destination rectangle, computed once per conflict rather than once per
// pair of clashing patches.
type patchBounds struct {
	r  image.Rectangle
	ok bool
}

func editBounds(hits []packHit) [][]patchBounds {
	out := make([][]patchBounds, len(hits))
	for i, hit := range hits {
		out[i] = make([]patchBounds, len(hit.edits))
		for j, edit := range hit.edits {
			out[i][j].r, out[i][j].ok = destBounds(edit)
		}
	}
	return out
}

func overlapAgainst(patch cpPatch, hits []packHit, bounds [][]patchBounds, self mod.ID) (x, y, w, h int, ok bool) {
	own, ok := destBounds(patch)
	if !ok {
		return 0, 0, 0, 0, false
	}
	var inter image.Rectangle
	found := false
	for i, hit := range hits {
		if mod.Equal(hit.id, self) {
			continue
		}
		for _, peer := range bounds[i] {
			if !peer.ok {
				continue
			}
			r := own.Intersect(peer.r)
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

func destBounds(p cpPatch) (image.Rectangle, bool) {
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
	for _, s := range p.shapes {
		if s.kind == 'p' || s.kind == 'w' {
			continue
		}
		if s.cells != "" {
			for cell := range strings.SplitSeq(s.cells, ";") {
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

func sourceCrop(p cpPatch, ox, oy, ow, oh int) (int, int, int, int) {
	fx, fy := 0, 0
	if from, ok := areaOf(json.RawMessage(p.imageFromArea)); ok {
		fx, fy = from.x, from.y
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
