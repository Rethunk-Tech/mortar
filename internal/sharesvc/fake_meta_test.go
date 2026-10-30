package sharesvc

import (
	"context"

	"github.com/Rethunk-Tech/go-nexusmods"

	"github.com/Rethunk-Tech/mortar/internal/deps"
	"github.com/Rethunk-Tech/mortar/internal/problems"
)

func (f fakeMeta) Collection(context.Context, string, string, int) (nexusmods.Collection, error) {
	return f.coll, f.collErr
}

var stardewEnv = problems.Environment{Nexus: nexusmods.Title{Domain: "stardewvalley", ID: 1303}, VersionScheme: deps.SemverSMAPI}
