package fomod

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"github.com/Rethunk-Tech/mortar/internal/fsx"
)

const configName = "moduleconfig.xml"

// maxConfigBytes bounds a ModuleConfig.xml from an archive; the largest real one is about 220 KB.
const maxConfigBytes = 16 << 20

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
	f, err := fsx.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	raw, err := fsx.ReadCapped(f, maxConfigBytes)
	if err != nil {
		return nil, err
	}
	return decodeXML(raw), nil
}

func decodeXML(raw []byte) []byte {
	if len(raw) >= 2 && raw[0] == 0xFF && raw[1] == 0xFE {
		return utf16ToUTF8(raw[2:], binary.LittleEndian)
	}
	if len(raw) >= 2 && raw[0] == 0xFE && raw[1] == 0xFF {
		return utf16ToUTF8(raw[2:], binary.BigEndian)
	}
	if len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF {
		return raw[3:]
	}
	// XML appendix F: a BOM-less UTF-16 document starts with an encoded "<?".
	switch {
	case bytes.HasPrefix(raw, []byte("<\x00?\x00")):
		return utf16ToUTF8(raw, binary.LittleEndian)
	case bytes.HasPrefix(raw, []byte("\x00<\x00?")):
		return utf16ToUTF8(raw, binary.BigEndian)
	}
	return raw
}

// utf16ToUTF8 decodes UTF-16 in order's byte order, dropping a trailing odd byte.
func utf16ToUTF8(b []byte, order binary.ByteOrder) []byte {
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = order.Uint16(b[i*2:])
	}
	return []byte(string(utf16.Decode(u)))
}
