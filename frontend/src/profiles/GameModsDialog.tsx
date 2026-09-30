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
  const importMods = async () => {
    if (busy) {
      return
    }
    setBusy(true)
    try {
      const res = await ImportGameMods(game)
      await load(game)
      useToasts.getState().push({
        kind: 'success',
        title: t`Imported ${res.profile.name}`,
        body: [
          plural(res.imported, { one: '# imported', other: '# imported' }),
          plural(res.skipped, { one: '# skipped', other: '# skipped' }),
          plural(res.failed, { one: '# failed', other: '# failed' }),
        ].join(' · '),
      })
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
              {plural(mods.length, {
                one: '# mod will be copied into a new profile. The game folder is left as it is.',
                other:
                  '# mods will be copied into a new profile. The game folder is left as it is.',
              })}
            </Typography>
            {mods.map((m) => (
              <Typography
                key={`${m.name}-${m.version}-${m.source}`}
                sx={{ fontSize: 14, py: 0.25 }}
              >
                {[m.name, m.version, m.source].join(' · ')}
              </Typography>
            ))}
          </>
        ) : (
          <Typography sx={{ color: 'error.main' }}>{error}</Typography>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Cancel`}</Button>
        <Button
          variant="contained"
          disabled={busy || mods.length === 0 || error !== ''}
          onClick={() => importMods().catch(reportUnexpected)}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t`Import`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
