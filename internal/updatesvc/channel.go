package updatesvc

import (
	"context"
	"io"

	"github.com/Rethunk-AI/mortar/internal/meta"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

const providerMetaKey = "mortar.provider"

type channelProvider struct {
	stable      updater.Provider
	beta        updater.Provider
	includeBeta func() bool
}

func (p *channelProvider) Name() string { return "mortar" }

func (p *channelProvider) Check(ctx context.Context, req updater.CheckRequest) (*updater.Release, error) {
	stable, err := p.stable.Check(ctx, req)
	if err != nil {
		return nil, err
	}
	if p.includeBeta == nil || !p.includeBeta() {
		return stampProvider(stable, p.stable.Name()), nil
	}
	beta, err := p.beta.Check(ctx, req)
	if err != nil {
		return stampProvider(stable, p.stable.Name()), err
	}
	chosen := preferRelease(true, stable, beta)
	if chosen == nil {
		return stable, nil
	}
	provider := p.stable.Name()
	if beta != nil && chosen.Version == beta.Version && meta.Newer(beta.Version, req.CurrentVersion) {
		provider = p.beta.Name()
	}
	return stampProvider(chosen, provider), nil
}

func stampProvider(rel *updater.Release, provider string) *updater.Release {
	if rel == nil {
		return nil
	}
	if rel.Metadata == nil {
		rel.Metadata = map[string]any{}
	}
	rel.Metadata[providerMetaKey] = provider
	return rel
}

func (p *channelProvider) Download(ctx context.Context, rel *updater.Release, dst io.Writer, onProgress func(written, total int64)) error {
	if rel != nil && rel.Metadata != nil && rel.Metadata[providerMetaKey] == p.beta.Name() {
		return p.beta.Download(ctx, rel, dst, onProgress)
	}
	return p.stable.Download(ctx, rel, dst, onProgress)
}
