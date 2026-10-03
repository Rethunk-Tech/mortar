package sharesvc

import (
	"context"

	"github.com/Rethunk-AI/mortar/internal/meta"
)

func (fakeMeta) PageRequirements(context.Context, int) ([]meta.Requirement, error) {
	return nil, nil
}

func (f fakeMeta) Collection(context.Context, string, string, int) (meta.Collection, error) {
	return f.coll, f.collErr
}
