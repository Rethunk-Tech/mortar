package devsvc

import (
	"errors"
	"strings"

	"github.com/Rethunk-Tech/mortar/internal/components"
	"github.com/Rethunk-Tech/mortar/internal/secret"
	"github.com/Rethunk-Tech/mortar/internal/usererr"
)

const (
	// curseForgeKeyring is the item an author's CurseForge upload token is kept under, one per user.
	curseForgeKeyring = "curseforge-upload"
	curseForgeName    = "CurseForge"
)

// curseForgeBase is the host a game's upload API answers on, empty when the catalog names none.
func curseForgeBase(gameID string) string {
	g, _ := components.Game(gameID)
	if src, ok := g.Source("curseforge"); ok && src.UploadSubdomain != "" {
		return "https://" + src.UploadSubdomain + ".curseforge.com"
	}
	return ""
}

// SetCurseForgeToken keeps the author's CurseForge upload token in the keyring.
func (s *Service) SetCurseForgeToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return usererr.Wrap(usererr.Invalid, errors.New("the token is empty"))
	}
	return secret.Set(curseForgeKeyring, token)
}

// HasCurseForgeToken reports whether an upload token is kept.
func (s *Service) HasCurseForgeToken() bool {
	token, err := secret.Get(curseForgeKeyring)
	return err == nil && token != ""
}

// ForgetCurseForgeToken removes the kept upload token.
func (s *Service) ForgetCurseForgeToken() error { return secret.Delete(curseForgeKeyring) }
