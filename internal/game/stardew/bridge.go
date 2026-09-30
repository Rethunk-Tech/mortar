package stardew

import (
	"crypto/sha256"
	_ "embed" // the bridge archive
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Rethunk-AI/mortar/internal/archive"
)

// Pinned by scripts/update-bridge.sh from github.com/Rethunk-AI/mortar-smapi-bridge, release
// bridgeVersion, asset MortarSmapiBridge-<version>.zip, sha256 bridgeSHA256.
const (
	bridgeVersion = "1.0.0"
	bridgeSHA256  = "e0286e859aac45755d4b1a7924472c87727cd98afd4855afb8b604fd74fa4675"
)

//go:embed vendor/MortarSmapiBridge-1.0.0.zip
var bridgeZip []byte

// BridgeVersion is the bundled Mortar SMAPI Bridge's version.
func (Game) BridgeVersion() string { return bridgeVersion }

// ExtractBridge unpacks the bundled bridge mod (a MortarSmapiBridge folder) into dst after checking the archive.
func (Game) ExtractBridge(dst string) error {
	sum := sha256.Sum256(bridgeZip)
	if got := hex.EncodeToString(sum[:]); got != bridgeSHA256 {
		return fmt.Errorf("bundled bridge archive has sha256 %s, want %s", got, bridgeSHA256)
	}
	work, err := os.MkdirTemp("", "mortar-bridge-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(work) }()
	zipPath := filepath.Join(work, "bridge.zip")
	if err := os.WriteFile(zipPath, bridgeZip, 0o600); err != nil {
		return err
	}
	return archive.Extract(zipPath, dst, archive.Options{})
}
