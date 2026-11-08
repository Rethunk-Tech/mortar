import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Button, Checkbox, Chip, FormControlLabel, Typography } from '@mui/material'
import { useEffect, useRef, useState } from 'react'
import type { GameModPreview } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  ImportGameMods,
  PreviewGameMods,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { modsLabel } from '../i18n/counts.ts'
import { lowerFirst } from '../i18n/lowerFirst.ts'
import { useReasonText } from '../i18n/reasonText.ts'
import { brokenKindLabel } from '../mods/brokenKind.ts'
import type { GameId } from '../nav/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { AppDialog, DialogFooter } from '../shell/dialog/AppDialog.tsx'
import { ErrorRetry } from '../shell/ErrorRetry.tsx'
import { RefreshFailed } from '../shell/RefreshFailed.tsx'
import { SkeletonRows } from '../shell/SkeletonRows.tsx'
import { useLastGood, useLastGoodRefreshing } from '../shell/useLastGood.ts'
import { type InlineError, inlineError, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { GameModsProgressLine } from './GameModsProgress.tsx'
import { formatOutcomeDetail, formatPreviewRow, willImport } from './gameModsFormat.ts'
import { useProfiles } from './store.ts'
import { useGameModsProgress } from './useGameModsProgress.ts'

const NO_MODS: GameModPreview[] = []

// What the import will do, and for a game that reads loose files from its folder the choice to move them out.
function Plan({
  count,
  canMove,
  move,
  busy,
  onMove,
}: {
  count: number
  canMove: boolean
  move: boolean
  busy: boolean
  onMove: (on: boolean) => void
}) {
  const { t } = useLingui()
  const mods = modsLabel(count)
  return (
    <>
      <Typography sx={{ mb: 1.5, color: 'text.secondary' }}>
        {canMove && move
          ? t`${mods} will be moved into a new profile and out of the game's Mods folder. Mortar puts the profile's mods there each time you play.`
          : t`${mods} will be copied into a new profile. The game folder is left as it is.`}
      </Typography>
      {canMove ? (
        <>
          <FormControlLabel
            control={
              <Checkbox
                size="small"
                checked={move}
                disabled={busy}
                onChange={(_, on) => onMove(on)}
              />
            }
            label={t`Move them out of the game's Mods folder`}
          />
          <Typography sx={{ mb: 1.5, fontSize: 'base.fontSize', color: 'text.secondary' }}>
            {t`If they stay, the game keeps loading them even when a profile disables them.`}
          </Typography>
        </>
      ) : null}
    </>
  )
}

function PreviewRows({ mods }: { mods: GameModPreview[] }) {
  const { t } = useLingui()
  const switchedOff = lowerFirst(t`Disabled`)
  const reasonText = useReasonText()
  return mods.map((m) => (
    <Typography
      key={`${m.name}-${m.version}-${m.source}-${m.status}-${m.reason}`}
      sx={{ fontSize: 'medium.fontSize', py: 0.25 }}
    >
      {formatPreviewRow(m, switchedOff, reasonText)}
      {m.broken ? (
        <Chip size="small" label={brokenKindLabel(m.brokenKind)} title={m.broken} sx={{ ml: 1 }} />
      ) : null}
    </Typography>
  ))
}

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
  const reasonText = useReasonText()
  const load = useProfiles((s) => s.load)
  const [move, setMove] = useState(false)
  const [importError, setImportError] = useState<InlineError | null>(null)
  const [busy, setBusy] = useState(false)
  const preview = useLastGood<{ mods: GameModPreview[]; canMove: boolean }>(
    `game-mods:${game}`,
    open
      ? () =>
          PreviewGameMods(game).then((p) => ({ mods: p.mods ?? [], canMove: p.canMove === true }))
      : null,
    [open],
  )
  const mods = preview.data?.mods ?? NO_MODS
  const canMove = preview.data?.canMove ?? false
  const stale = preview.data === null ? null : preview.error
  const error = importError ?? (preview.data === null ? preview.error : null)
  const previewing = open && preview.data === null && preview.error === null
  // Nothing imports from a list the live folder has not confirmed.
  const rechecking =
    useLastGoodRefreshing(open ? `game-mods:${game}` : null) && preview.data !== null
  const loadPreview = () => {
    preview.reload().catch(reportUnexpected)
  }
  const progress = useGameModsProgress(game, busy)
  const gen = useRef(0)
  const live = useRef(false)
  useEffect(() => {
    if (!open) {
      live.current = false
      gen.current += 1
      setMove(false)
      setImportError(null)
      setBusy(false)
      return
    }
    live.current = true
  }, [open])
  const importable = mods.filter((m) => willImport(m.status)).length
  const importMods = async () => {
    if (busy || previewing || rechecking || !live.current) {
      return
    }
    const token = gen.current
    setBusy(true)
    try {
      const res = await ImportGameMods(game, canMove && move)
      if (token !== gen.current || !live.current) {
        return
      }
      await load(game)
      if (token !== gen.current || !live.current) {
        return
      }
      const detail = formatOutcomeDetail(res.outcomes ?? [], reasonText)
      const toast = {
        kind: 'success' as const,
        title: t`Created ${{ name: res.profile.name }}`,
        body: [
          plural({ count: res.imported }, { one: '# imported', other: '# imported' }),
          plural({ count: res.skipped }, { one: '# skipped', other: '# skipped' }),
          plural({ count: res.failed }, { one: '# failed', other: '# failed' }),
        ].join(' · '),
      }
      useToasts.getState().push(detail === '' ? toast : { ...toast, detail })
      onImported(res.profile.id)
      onClose()
    } catch (e) {
      if (token !== gen.current || !live.current) {
        return
      }
      setImportError(inlineError(e))
    } finally {
      if (token === gen.current) {
        setBusy(false)
      }
    }
  }
  return (
    <AppDialog
      open={open}
      onClose={busy ? () => undefined : onClose}
      title={t`Import from the game's Mods folder`}
      footer={
        <DialogFooter>
          <Button onClick={onClose} disabled={busy}>
            {t`Cancel`}
          </Button>
          <DisabledReason
            title={t`Nothing to import`}
            disabled={previewing || rechecking || importable === 0}
          >
            <Button
              variant="contained"
              disabled={
                busy ||
                previewing ||
                rechecking ||
                importable === 0 ||
                error !== null ||
                stale !== null
              }
              onClick={() => importMods().catch(reportUnexpected)}
            >
              {t`Import`}
            </Button>
          </DisabledReason>
        </DialogFooter>
      }
    >
      {error === null ? null : <ErrorRetry error={error} onRetry={loadPreview} />}
      {error === null && previewing ? (
        <SkeletonRows label={t`Reading the Mods folder…`} count={5} height={36} />
      ) : null}
      {error === null && rechecking && !busy ? (
        <Typography role="status" sx={{ color: 'text.secondary' }}>
          {t`Rechecking the folder…`}
        </Typography>
      ) : null}
      {busy ? <GameModsProgressLine progress={progress} moving={canMove && move} /> : null}
      {error === null && !previewing ? (
        <>
          <RefreshFailed error={stale} at={preview.at} onRetry={loadPreview} />
          <Plan count={importable} canMove={canMove} move={move} busy={busy} onMove={setMove} />
          <PreviewRows mods={mods} />
        </>
      ) : null}
    </AppDialog>
  )
}
