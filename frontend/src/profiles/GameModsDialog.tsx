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
import { useCallback, useEffect, useRef, useState } from 'react'
import type { GameModPreview } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  ImportGameMods,
  PreviewGameMods,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import type { GameId } from '../nav/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { type InlineError, inlineError, reportUnexpected } from '../toasts/report.ts'
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
  const [error, setError] = useState<InlineError | null>(null)
  const [busy, setBusy] = useState(false)
  const [previewing, setPreviewing] = useState(false)
  const gen = useRef(0)
  const live = useRef(false)
  const loadPreview = useCallback(() => {
    gen.current += 1
    const token = gen.current
    setMods([])
    setError(null)
    setPreviewing(true)
    PreviewGameMods(game)
      .then((p) => {
        if (token !== gen.current || !live.current) {
          return
        }
        setMods(p.mods ?? [])
        setPreviewing(false)
      })
      .catch((e: unknown) => {
        if (token !== gen.current || !live.current) {
          return
        }
        setMods([])
        setError(inlineError(e))
        setPreviewing(false)
      })
  }, [game])
  useEffect(() => {
    if (!open) {
      live.current = false
      gen.current += 1
      setMods([])
      setError(null)
      setBusy(false)
      setPreviewing(false)
      return
    }
    live.current = true
    loadPreview()
  }, [open, loadPreview])
  const importable = mods.filter((m) => willImport(m.status)).length
  const importMods = async () => {
    if (busy || previewing || !live.current) {
      return
    }
    const token = gen.current
    setBusy(true)
    try {
      const res = await ImportGameMods(game)
      if (token !== gen.current || !live.current) {
        return
      }
      await load(game)
      if (token !== gen.current || !live.current) {
        return
      }
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
      if (token !== gen.current || !live.current) {
        return
      }
      setError(inlineError(e))
    } finally {
      if (token === gen.current) {
        setBusy(false)
      }
    }
  }
  return (
    <Dialog
      open={open}
      onClose={busy ? undefined : onClose}
      slotProps={{ paper: { sx: { minWidth: 420, maxWidth: 'calc(100vw - 64px)' } } }}
    >
      <DialogTitle>{t`Import from the game's Mods folder`}</DialogTitle>
      <DialogContent>
        {error === null ? null : (
          <>
            <Typography sx={{ color: 'error.main' }} title={error.details}>
              {error.message}
            </Typography>
            <Button onClick={loadPreview} sx={{ mt: 1 }}>
              {t`Retry`}
            </Button>
          </>
        )}
        {error === null && previewing ? (
          <LoadingRow>{t`Reading the Mods folder…`}</LoadingRow>
        ) : null}
        {error === null && !previewing ? (
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
        ) : null}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={busy}>
          {t`Cancel`}
        </Button>
        <DisabledReason title={t`Nothing to import`} disabled={previewing || importable === 0}>
          <Button
            variant="contained"
            disabled={busy || previewing || importable === 0 || error !== null}
            onClick={() => importMods().catch(reportUnexpected)}
          >
            {t`Import`}
          </Button>
        </DisabledReason>
      </DialogActions>
    </Dialog>
  )
}
