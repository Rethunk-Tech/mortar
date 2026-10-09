package sharesvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/profile"
	"github.com/Rethunk-Tech/mortar/internal/share"
)

func TestAPatreonModIsUnavailableUntilTheReceiverHoldsAFileFromThePost(t *testing.T) {
	ref := share.Ref{Patreon: "4242"}
	m := (&resolver{}).patreon(ref)
	if m.Site != SitePatreon || m.State != StateUnavailable || m.Reason != ReasonPatreon || m.PageURL != "https://www.patreon.com/posts/4242" {
		t.Fatalf("mod = %+v", m)
	}
	held := &resolver{target: []profile.Entry{{Source: profile.Source{Kind: profile.KindPatreon, Name: "4242"}}}}
	if got := held.patreon(ref); got.State != StateInstalled {
		t.Fatalf("a profile that holds the post's file has it installed: %+v", got)
	}
}
