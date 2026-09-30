import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from '@mui/material'
import { useEffect, useState } from 'react'
import type { GameModPreview } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  ImportGameMods,
  PreviewGameMods,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import type { GameId } from '../nav/store.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { formatOutcomeDetail, formatPreviewRow, willImport } from './gameModsFormat.ts'
import { useProfiles } from './store.ts'

export function GameModsDialog({
  open,
  game,
  onClose,
  onImported,
}: {
  open: boolean
  game: GameId
  onClose: () => void
  onImported: (id: string) => void
}) {
  const { t } = useLingui()
  const switchedOff = t`switched off`
  const load = useProfiles((s) => s.load)
  const [mods, setMods] = useState<GameModPreview[]>([])
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  useEffect(() => {
    if (!open) {
      return
    }
    setError('')
    PreviewGameMods(game)
      .then((p) => setMods(p.mods ?? []))
      .catch((e: unknown) => {
        setMods([])
        setError(errorMessage(e))
      })
  }, [open, game])
  const importable = mods.filter((m) => willImport(m.status)).length
  const importMods = async () => {
    if (busy) {
      return
    }
    setBusy(true)
    try {
      const res = await ImportGameMods(game)
      await load(game)
      const detail = formatOutcomeDetail(res.outcomes ?? [])
      const toast = {
        kind: 'success' as const,
        title: t`Created “${res.profile.name}”`,
        body: [
          plural(res.imported, { one: '# imported', other: '# imported' }),
          plural(res.skipped, { one: '# skipped', other: '# skipped' }),
          plural(res.failed, { one: '# failed', other: '# failed' }),
        ].join(' · '),
      }
      useToasts.getState().push(detail === '' ? toast : { ...toast, detail })
      onImported(res.profile.id)
      onClose()
    } catch (e) {
      setError(errorMessage(e))
    } finally {
      setBusy(false)
    }
  }
  return (
    <Dialog
      open={open}
      onClose={onClose}
      transitionDuration={0}
      slotProps={{ paper: { sx: { bgcolor: 'rgb(40,40,48)', minWidth: 420 } } }}
    >
      <DialogTitle>{t`Import from the game's Mods folder`}</DialogTitle>
      <DialogContent>
        {error === '' ? (
          <>
            <Typography sx={{ mb: 1.5, color: 'text.secondary' }}>
              {plural(importable, {
                one: '# mod will be copied into a new profile. The game folder is left as it is.',
                other:
                  '# mods will be copied into a new profile. The game folder is left as it is.',
              })}
            </Typography>
            {mods.map((m) => (
              <Typography
                key={`${m.name}-${m.version}-${m.source}-${m.status}-${m.reason}`}
                sx={{ fontSize: 14, py: 0.25 }}
              >
                {formatPreviewRow(m, switchedOff)}
              </Typography>
            ))}
          </>
        ) : (
          <Typography sx={{ color: 'error.main' }}>{error}</Typography>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} sx={{ whiteSpace: 'nowrap' }}>
          {t`Cancel`}
        </Button>
        <Button
          variant="contained"
          disabled={busy || importable === 0 || error !== ''}
          onClick={() => importMods().catch(reportUnexpected)}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t`Import`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
