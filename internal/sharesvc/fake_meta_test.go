package sharesvc

import (
	"context"

	"github.com/Rethunk-Tech/mortar/internal/meta"
	"github.com/Rethunk-Tech/mortar/internal/nexus"
	"github.com/Rethunk-Tech/mortar/internal/problems"
)

func (fakeMeta) PageRequirements(context.Context, string, int) ([]meta.Requirement, error) {
	return nil, nil
}

func (f fakeMeta) Collection(context.Context, string, string, int) (meta.Collection, error) {
	return f.coll, f.collErr
}

var stardewEnv = problems.Environment{Nexus: nexus.Title{Domain: "stardewvalley", ID: 1303}}
