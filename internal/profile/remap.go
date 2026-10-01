package profile

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Rethunk-AI/mortar/internal/fsx"
	"github.com/Rethunk-AI/mortar/internal/manifest"
	"github.com/Rethunk-AI/mortar/internal/store"
)

// RemapAsk is a dialog so the user can pick which extracted folder is the mod.
type RemapAsk struct {
	Key    string      `json:"key"`
	Source Source      `json:"source"`
	Tree   []RemapNode `json:"tree"`
}

// RemapNode is one file or folder in the extracted archive.
type RemapNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	Dir      bool        `json:"dir"`
	Size     int64       `json:"size"`
	Children []RemapNode `json:"children,omitempty"`
}

// NeedRootError means the store item has no SMAPI-reachable manifest and the user has not chosen a folder.
type NeedRootError struct {
	Ask RemapAsk
}

func (e *NeedRootError) Error() string { return "this archive needs a mod folder chosen" }

func (s *Store) remapAsk(game, key string) (RemapAsk, bool, error) {
	dir, err := s.items.Path(game, key)
	if err != nil {
		return RemapAsk{}, false, err
	}
	rel, err := s.items.Root(game, key)
	if err != nil {
		return RemapAsk{}, false, err
	}
	if rel != "" {
		return RemapAsk{}, false, nil
	}
	if leftover := contentRootFile(dir); leftover != "" {
		tree, err := remapTree(dir)
		if err != nil {
			return RemapAsk{}, false, err
		}
		return RemapAsk{Key: key, Tree: tree}, true, nil
	}
	found, err := manifest.Scan(dir)
	if err != nil {
		return RemapAsk{}, false, err
	}
	if len(found) > 0 {
		return RemapAsk{}, false, nil
	}
	tree, err := remapTree(dir)
	if err != nil {
		return RemapAsk{}, false, err
	}
	return RemapAsk{Key: key, Tree: tree}, true, nil
}

func contentRootFile(dir string) string {
	b, err := fsx.ReadFile(filepath.Join(dir, store.RootFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func remapTree(root string) ([]RemapNode, error) {
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []RemapNode
	for _, e := range ents {
		if e.Name() == store.RootFile || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		n, err := remapNode(root, e.Name(), e)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

func remapNode(root, rel string, e fs.DirEntry) (RemapNode, error) {
	p := filepath.Join(root, filepath.FromSlash(rel))
	n := RemapNode{Name: e.Name(), Path: filepath.ToSlash(rel), Dir: e.IsDir()}
	if !e.IsDir() {
		info, err := e.Info()
		if err != nil {
			return RemapNode{}, err
		}
		n.Size = info.Size()
		return n, nil
	}
	children, err := os.ReadDir(p)
	if err != nil {
		return RemapNode{}, err
	}
	for _, c := range children {
		if c.Name() == store.RootFile || strings.HasPrefix(c.Name(), ".") {
			continue
		}
		next := c.Name()
		if rel != "" {
			next = rel + "/" + c.Name()
		}
		ch, err := remapNode(root, next, c)
		if err != nil {
			return RemapNode{}, err
		}
		n.Children = append(n.Children, ch)
		n.Size += ch.Size
	}
	return n, nil
}

// InstallRemap records the chosen folder as this store item's root and adds the item to the profile.
func (s *Store) InstallRemap(game, id, key, root string, source Source) (InstallResult, error) {
	if err := s.unlocked(game, id); err != nil {
		return InstallResult{}, err
	}
	dir, err := s.items.Path(game, key)
	if err != nil {
		return InstallResult{}, installError(err)
	}
	rel := path.Clean("/" + strings.TrimSpace(strings.ReplaceAll(root, `\`, "/")))
	rel = strings.TrimPrefix(rel, "/")
	if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
		return InstallResult{}, &InstallError{Msg: "Choose a folder that holds a SMAPI manifest", Err: fmt.Errorf("root %q", root)}
	}
	sub := filepath.Join(dir, filepath.FromSlash(rel))
	found, err := manifest.Scan(sub)
	if err != nil {
		return InstallResult{}, installError(err)
	}
	if len(found) == 0 {
		return InstallResult{}, &InstallError{Msg: "Choose a folder that holds a SMAPI manifest", Err: &NoModError{Key: key}}
	}
	if err := s.items.SetRoot(game, key, rel); err != nil {
		return InstallResult{}, installError(err)
	}
	return s.installKey(game, id, key, source)
}

func (s *Service) InstallRemap(game, id, key, root string, source Source) (InstallResult, error) {
	return s.store.InstallRemap(game, id, key, root, source)
}
