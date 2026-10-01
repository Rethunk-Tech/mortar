package fomod

import (
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/Rethunk-AI/mortar/internal/fsx"
)

const configName = "moduleconfig.xml"

// FindConfig returns the path of fomod/ModuleConfig.xml under root.
// The fomod directory may sit at root or one level down; names are matched
// case-insensitively.
func FindConfig(root string) (string, error) {
	return findAtDepth(root, 0)
}

func findAtDepth(dir string, depth int) (string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var fomodDir string
	for _, e := range ents {
		if e.IsDir() && strings.EqualFold(e.Name(), "fomod") {
			fomodDir = filepath.Join(dir, e.Name())
			break
		}
	}
	if fomodDir != "" {
		fents, err := os.ReadDir(fomodDir)
		if err != nil {
			return "", err
		}
		for _, e := range fents {
			if !e.IsDir() && strings.EqualFold(e.Name(), configName) {
				return filepath.Join(fomodDir, e.Name()), nil
			}
		}
	}
	if depth >= 1 {
		return "", nil
	}
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		if strings.EqualFold(e.Name(), "fomod") {
			continue
		}
		p, err := findAtDepth(filepath.Join(dir, e.Name()), depth+1)
		if err != nil {
			return "", err
		}
		if p != "" {
			return p, nil
		}
	}
	return "", nil
}

func readXMLBytes(path string) ([]byte, error) {
	raw, err := fsx.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decodeXML(raw), nil
}

func decodeXML(raw []byte) []byte {
	if len(raw) >= 2 && raw[0] == 0xFF && raw[1] == 0xFE {
		return utf16LEToUTF8(raw[2:])
	}
	if len(raw) >= 2 && raw[0] == 0xFE && raw[1] == 0xFF {
		return utf16BEToUTF8(raw[2:])
	}
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		return raw[3:]
	}
	if len(raw) >= 2 && raw[1] == 0 && !utf8.Valid(raw) {
		return utf16LEToUTF8(raw)
	}
	return raw
}

func utf16LEToUTF8(b []byte) []byte {
	if len(b)%2 == 1 {
		b = b[:len(b)-1]
	}
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = uint16(b[i*2]) | uint16(b[i*2+1])<<8
	}
	return []byte(string(utf16.Decode(u)))
}

func utf16BEToUTF8(b []byte) []byte {
	if len(b)%2 == 1 {
		b = b[:len(b)-1]
	}
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = uint16(b[i*2+1]) | uint16(b[i*2])<<8
	}
	return []byte(string(utf16.Decode(u)))
}
