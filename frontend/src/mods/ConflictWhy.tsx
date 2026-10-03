import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { type ReactNode, useEffect, useState } from 'react'
import type {
  AssetConflict,
  ConflictEvidence,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { ConflictImageCrop } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'

function previewKey(e: ConflictEvidence) {
  return `${e.packId}:${e.fromFile ?? ''}:${e.cropX},${e.cropY},${e.cropW}x${e.cropH}`
}

function evidenceKey(e: ConflictEvidence) {
  return `${e.packId}:${e.source}:${e.index}:${e.action}:${e.target}`
}

const KEYS_SHOWN = 8

function EvidenceLine({ children }: { children: ReactNode }) {
  return (
    <Typography sx={{ fontSize: 12, color: 'text.secondary', whiteSpace: 'normal' }}>
      {children}
    </Typography>
  )
}

export function ConflictWhy({ asset }: { asset: AssetConflict }) {
  const { t } = useLingui()
  const gameId = useProfiles((s) => s.game?.id) ?? ''
  const openId = useProfiles((s) => s.openId)
  const [previews, setPreviews] = useState<Record<string, string>>({})
  useEffect(() => {
    let cancelled = false
    const crops = (asset.evidence ?? []).filter(
      (e) => e.cropW > 0 && (e.fromFile ?? '') !== '' && e.packId !== '',
    )
    if (crops.length === 0 || gameId === '' || openId === '') {
      return () => {
        cancelled = true
      }
    }
    Promise.all(
      crops.map((e) =>
        ConflictImageCrop(
          gameId,
          openId,
          e.packId,
          e.fromFile ?? '',
          e.cropX,
          e.cropY,
          e.cropW,
          e.cropH,
        )
          .then((url) => ({ key: previewKey(e), url }))
          .catch(() => ({ key: previewKey(e), url: '' })),
      ),
    )
      .then((rows) => {
        if (cancelled) {
          return
        }
        const next: Record<string, string> = {}
        for (const row of rows) {
          if (row.url !== '') {
            next[row.key] = row.url
          }
        }
        setPreviews(next)
      })
      .catch(reportUnexpected)
    return () => {
      cancelled = true
    }
  }, [asset, gameId, openId])
  const images = (asset.evidence ?? []).filter((e) => previews[previewKey(e)])
  return (
    <Box sx={{ mt: 0.75, display: 'flex', flexDirection: 'column', gap: 1 }}>
      {(asset.evidence ?? []).map((e) => (
        <Box key={evidenceKey(e)}>
          <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{e.packName || e.packId}</Typography>
          <EvidenceLine>{t`${e.source} · ${e.index}`}</EvidenceLine>
          <EvidenceLine>{t`${e.action} ${e.target}`}</EvidenceLine>
          {e.keys && e.keys.length > 0 ? (
            <Typography
              title={e.keys.join('\n')}
              sx={{
                fontSize: 12,
                color: 'warning.main',
                whiteSpace: 'normal',
                overflowWrap: 'anywhere',
              }}
            >
              {e.keys.length > KEYS_SHOWN
                ? t`Both set: ${e.keys.slice(0, KEYS_SHOWN).join(', ')} +${e.keys.length - KEYS_SHOWN} more`
                : t`Both set: ${e.keys.join(', ')}`}
            </Typography>
          ) : null}
          {e.toArea ? <EvidenceLine>{t`ToArea ${e.toArea}`}</EvidenceLine> : null}
          {e.fromArea ? <EvidenceLine>{t`FromArea ${e.fromArea}`}</EvidenceLine> : null}
          {e.when ? <EvidenceLine>{t`When ${e.when}`}</EvidenceLine> : null}
          <EvidenceLine>
            {e.priority === '' ? t`Priority Default` : t`Priority ${e.priority}`}
          </EvidenceLine>
        </Box>
      ))}
      {images.length === 0 ? null : (
        <Box sx={{ display: 'flex', flexDirection: 'row', gap: 1, flexWrap: 'wrap' }}>
          {images.map((e) => {
            const src = previews[previewKey(e)]
            return (
              <Box key={previewKey(e)} sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
                  {e.packName || e.packId}
                </Typography>
                <Box
                  component="img"
                  src={src}
                  alt={e.packName || e.packId}
                  sx={{
                    imageRendering: 'pixelated',
                    maxWidth: 128,
                    maxHeight: 128,
                    border: '1px solid var(--mortar-hairline-20)',
                  }}
                />
              </Box>
            )
          })}
        </Box>
      )}
    </Box>
  )
}
