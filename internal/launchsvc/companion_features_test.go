//go:build !windows

package launchsvc

import (
	"testing"

	"github.com/Rethunk-Tech/mortar/internal/bridge"
	"github.com/Rethunk-Tech/mortar/internal/game"
)

// What the companion can do is read from the features its status advertises. Before one arrives, and for a companion
// from before the split that advertises none, Mortar assumes the 0.5.x set rather than refusing a working feature.
func TestCompanionFeaturesDecideWhatMortarAsksOfIt(t *testing.T) {
	svc, p := startEnv(t)
	g := game.Find("stardew")
	if !svc.companionServes(g, p.ID, bridge.FeatureConsole) {
		t.Fatal("a companion that has not answered yet is taken as unable to take commands")
	}
	key := keyOf(g)
	svc.mu.Lock()
	svc.logs[key] = session{profile: p.ID, features: bridge.Features{bridge.FeatureConsole}}
	svc.mu.Unlock()
	if !svc.companionServes(g, p.ID, bridge.FeatureConsole) || svc.companionServes(g, p.ID, bridge.FeaturePerf) {
		t.Fatal("the advertised features were not followed")
	}
	// A status with no features is the 0.5.x companion, which served every feature there was.
	svc.mu.Lock()
	svc.logs[key] = session{profile: p.ID}
	svc.mu.Unlock()
	if !svc.companionServes(g, p.ID, bridge.FeaturePerf) {
		t.Fatal("a companion that advertises no features lost the 0.5.x set")
	}
}
