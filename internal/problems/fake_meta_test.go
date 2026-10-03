package problems

import (
	"context"

	"github.com/Rethunk-AI/mortar/internal/meta"
)

func (fakeMeta) PageRequirements(context.Context, int) ([]meta.Requirement, error) {
	return nil, nil
}

func (fakeMeta) Collection(context.Context, string, string, int) (meta.Collection, error) {
	return meta.Collection{}, nil
}
