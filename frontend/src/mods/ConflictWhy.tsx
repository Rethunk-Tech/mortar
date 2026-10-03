import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
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
          <Typography sx={{ fontSize: 12, color: 'text.secondary', whiteSpace: 'normal' }}>
            {t`${e.source} · ${e.index}`}
          </Typography>
          <Typography sx={{ fontSize: 12, color: 'text.secondary', whiteSpace: 'normal' }}>
            {t`${e.action} ${e.target}`}
          </Typography>
          {e.toArea ? (
            <Typography sx={{ fontSize: 12, color: 'text.secondary', whiteSpace: 'normal' }}>
              {t`ToArea ${e.toArea}`}
            </Typography>
          ) : null}
          {e.fromArea ? (
            <Typography sx={{ fontSize: 12, color: 'text.secondary', whiteSpace: 'normal' }}>
              {t`FromArea ${e.fromArea}`}
            </Typography>
          ) : null}
          {e.when ? (
            <Typography sx={{ fontSize: 12, color: 'text.secondary', whiteSpace: 'normal' }}>
              {t`When ${e.when}`}
            </Typography>
          ) : null}
          <Typography sx={{ fontSize: 12, color: 'text.secondary', whiteSpace: 'normal' }}>
            {e.priority === '' ? t`Priority Default` : t`Priority ${e.priority}`}
          </Typography>
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
                    border: '1px solid rgba(255,255,255,0.2)',
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
