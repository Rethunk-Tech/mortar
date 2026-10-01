package sharesvc

import (
	"context"

	"github.com/Rethunk-AI/mortar/internal/meta"
)

func (fakeMeta) PageRequirements(context.Context, int) ([]meta.Requirement, error) {
	return nil, nil
}
