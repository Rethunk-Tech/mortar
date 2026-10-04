package fomod

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/datadir"
)

// Apply copies ops from srcRoot into destRoot, later (higher priority) ops overwriting earlier ones.
func Apply(srcRoot, destRoot string, ops []Op) error {
	if err := os.MkdirAll(destRoot, 0o700); err != nil {
		return err
	}
	for _, op := range ops {
		if err := applyOp(srcRoot, destRoot, op); err != nil {
			return err
		}
	}
	return nil
}

func applyOp(srcRoot, destRoot string, op Op) error {
	from, err := under(srcRoot, op.Source)
	if err != nil {
		return err
	}
	dest := op.Destination
	if dest == "" {
		dest = filepath.Base(filepath.Clean(filepath.FromSlash(strings.ReplaceAll(op.Source, "\\", "/"))))
	}
	to, err := under(destRoot, dest)
	if err != nil {
		return err
	}
	info, err := os.Stat(from)
	if err != nil {
		return err
	}
	if op.Folder || info.IsDir() {
		if err := os.MkdirAll(to, 0o700); err != nil {
			return err
		}
		return copyOverwrite(from, to)
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o700); err != nil {
		return err
	}
	_ = os.Remove(to)
	return datadir.CopyFile(from, to)
}

func copyOverwrite(src, dst string) error {
	ents, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range ents {
		from := filepath.Join(src, e.Name())
		to := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := os.MkdirAll(to, 0o700); err != nil {
				return err
			}
			if err := copyOverwrite(from, to); err != nil {
				return err
			}
			continue
		}
		_ = os.Remove(to)
		if err := datadir.CopyFile(from, to); err != nil {
			return err
		}
	}
	return nil
}

func under(root, rel string) (string, error) {
	rel = filepath.FromSlash(strings.ReplaceAll(rel, "\\", "/"))
	if rel == "" || rel == "." {
		return root, nil
	}
	if !filepath.IsLocal(rel) {
		return "", fmt.Errorf("fomod path %q leaves the item", rel)
	}
	return filepath.Join(root, rel), nil
}

// Load parses ModuleConfig.xml at path.
func Load(path string) (Config, error) {
	b, err := readXMLBytes(path)
	if err != nil {
		return Config{}, err
	}
	return Parse(b)
}

// Open finds and parses a config under root. ok is false when none is there.
func Open(root string) (Config, string, bool, error) {
	path, err := FindConfig(root)
	if err != nil || path == "" {
		return Config{}, "", false, err
	}
	cfg, err := Load(path)
	return cfg, path, err == nil, err
}
