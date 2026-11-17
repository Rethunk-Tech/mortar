package bethesda

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/loader"
)

const (
	flagESM = 0x00000001
	flagESL = 0x00000200
)

type plugin struct {
	Name    string
	Masters []string
	ESM     bool
	ESL     bool
}

func parsePlugin(path string) (plugin, error) {
	f, err := os.Open(path)
	if err != nil {
		return plugin{}, err
	}
	defer f.Close()
	hdr := make([]byte, 24)
	if _, err := io.ReadFull(f, hdr); err != nil {
		return plugin{}, err
	}
	if string(hdr[:4]) != "TES4" {
		return plugin{}, fmt.Errorf("%s is not a TES4 plugin", path)
	}
	dataSize := binary.LittleEndian.Uint32(hdr[4:8])
	flags := binary.LittleEndian.Uint32(hdr[8:12])
	data := make([]byte, dataSize)
	if _, err := io.ReadFull(f, data); err != nil {
		return plugin{}, err
	}
	pl := plugin{ESM: flags&flagESM != 0, ESL: flags&flagESL != 0}
	for rest := data; len(rest) >= 6; {
		kind := string(rest[:4])
		n := int(binary.LittleEndian.Uint16(rest[4:6]))
		rest = rest[6:]
		if n > len(rest) {
			break
		}
		body := rest[:n]
		rest = rest[n:]
		if kind == "MAST" {
			pl.Masters = append(pl.Masters, string(bytes.TrimRight(body, "\x00")))
		}
	}
	return pl, nil
}

func lootOrder(plugins []plugin) ([]plugin, []loader.Finding) {
	byName := map[string]plugin{}
	for _, p := range plugins {
		byName[strings.ToLower(p.Name)] = p
	}
	var findings []loader.Finding
	deps := map[string][]string{}
	for _, p := range plugins {
		key := strings.ToLower(p.Name)
		for _, m := range p.Masters {
			mk := strings.ToLower(m)
			if _, ok := byName[mk]; !ok {
				findings = append(findings, missingFinding(p.Name, m))
				continue
			}
			deps[key] = append(deps[key], mk)
		}
	}
	const (
		white = iota
		grey
		black
	)
	color := map[string]int{}
	var order []string
	var visit func(string) bool
	visit = func(id string) bool {
		switch color[id] {
		case grey:
			return false
		case black:
			return true
		}
		color[id] = grey
		for _, d := range deps[id] {
			if !visit(d) {
				return false
			}
		}
		color[id] = black
		order = append(order, id)
		return true
	}
	ids := make([]string, 0, len(plugins))
	for _, p := range plugins {
		ids = append(ids, strings.ToLower(p.Name))
	}
	sortPlugins(ids, byName)
	for _, id := range ids {
		if color[id] != white {
			continue
		}
		if !visit(id) {
			var cycle []string
			for _, p := range plugins {
				if color[strings.ToLower(p.Name)] == grey {
					cycle = append(cycle, p.Name)
				}
			}
			findings = append(findings, cycleFinding(cycle))
			color[id] = black
			order = append(order, id)
		}
	}
	out := make([]plugin, 0, len(order))
	seen := map[string]bool{}
	for _, id := range order {
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, byName[id])
	}
	return out, findings
}

func sortPlugins(ids []string, byName map[string]plugin) {
	less := func(a, b string) bool {
		pa, pb := byName[a], byName[b]
		if pa.ESM != pb.ESM {
			return pa.ESM
		}
		if pa.ESL != pb.ESL {
			return pa.ESL
		}
		return strings.ToLower(pa.Name) < strings.ToLower(pb.Name)
	}
	for i := 1; i < len(ids); i++ {
		for j := i; j > 0 && less(ids[j], ids[j-1]); j-- {
			ids[j], ids[j-1] = ids[j-1], ids[j]
		}
	}
}
