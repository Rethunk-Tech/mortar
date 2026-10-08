import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Typography,
} from '@mui/material'
import { type ReactNode, useState } from 'react'
import type {
  GameModsDiff,
  GameModsResult,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { SyncGameModsFolders } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { GameModsProgressLine } from '../profiles/GameModsProgress.tsx'
import { formatOutcomeDetail } from '../profiles/gameModsFormat.ts'
import { useProfiles } from '../profiles/store.ts'
import { useGameModsProgress } from '../profiles/useGameModsProgress.ts'
import { space } from '../theme/density.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { LockedReason } from './LockedReason.tsx'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

const Section = ({ title, children }: { title: string; children: ReactNode }) => (
  <Box sx={{ mb: 1.5 }}>
    <Typography sx={{ fontWeight: 600, fontSize: 14, mb: 0.5 }}>{title}</Typography>
    {children}
  </Box>
)

function ReviewLists({
  diff,
  off,
  onToggle,
}: {
  diff: GameModsDiff
  off: ReadonlySet<string>
  onToggle: (id: string) => void
}) {
  const { t } = useLingui()
  const missing = diff.missing ?? []
  const different = diff.different ?? []
  const unreadable = diff.unreadable ?? []
  return (
    <>
      {missing.length === 0 ? null : (
        <Section title={t`Not in this profile (${missing.length})`}>
          {missing.map((m) => (
            <FormControlLabel
              key={m.id}
              sx={{ display: 'flex', overflowWrap: 'anywhere' }}
              control={
                <Checkbox checked={!off.has(m.id ?? '')} onChange={() => onToggle(m.id ?? '')} />
              }
              label={[m.name, m.version].filter((p) => p !== undefined && p !== '').join(' · ')}
            />
          ))}
        </Section>
      )}
      {different.length === 0 ? null : (
        <Section title={t`Different version (${different.length})`}>
          {different.map((m) => (
            <FormControlLabel
              key={m.id}
              sx={{ display: 'flex', overflowWrap: 'anywhere' }}
              control={<Checkbox checked={!off.has(m.id)} onChange={() => onToggle(m.id)} />}
              label={
                <>
                  {m.name}
                  <Typography component="span" sx={{ fontSize: 13, color: 'text.secondary' }}>
                    {` · ${t`folder ${m.folderVersion} · profile ${m.profileVersion}`}`}
                    {m.newer === 'folder' ? ` · ${t`newer in folder`}` : null}
                    {m.newer === 'profile' ? ` · ${t`newer in profile`}` : null}
                  </Typography>
                </>
              }
            />
          ))}
        </Section>
      )}
      {unreadable.length === 0 ? null : (
        <Section title={t`Could not read (${unreadable.length})`}>
          {unreadable.map((m) => (
            <Typography
              key={m.folder}
              sx={{ fontSize: 14, color: 'text.secondary', pl: space.gap }}
            >
              {`${m.name} · ${m.reason ?? ''}`}
            </Typography>
          ))}
        </Section>
      )}
      {diff.same > 0 || diff.profileOnly > 0 ? (
        <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
          {[
            diff.same > 0
              ? plural(diff.same, { one: '# mod matches.', other: '# mods match.' })
              : '',
            diff.profileOnly > 0
              ? plural(diff.profileOnly, {
                  one: '# profile mod has no copy in the game folder.',
                  other: '# profile mods have no copy in the game folder.',
                })
              : '',
          ]
            .filter(Boolean)
            .join(' ')}
        </Typography>
      ) : null}
    </>
  )
}

/** What the game's Mods folder holds that this profile lacks or has at another version, copied or moved in on request. */
export function GameModsReviewDialog({
  game,
  profile,
  diff,
  onClose,
}: {
  game: string
  profile: Profile
  /** The diff as it stood when the dialog opened; the dialog is mounted per opening. */
  diff: GameModsDiff
  onClose: () => void
}) {
  const { t } = useLingui()
  const locked = useLocked()
  const [off, setOff] = useState<ReadonlySet<string>>(
    () => new Set((diff.different ?? []).filter((m) => m.newer !== 'folder').map((m) => m.id)),
  )
  const [result, setResult] = useState<GameModsResult | null>(null)
  const [moving, setMoving] = useState(false)
  const [busy, run] = usePending()
  const progress = useGameModsProgress(game, busy)
  const missing = diff.missing ?? []
  const different = diff.different ?? []
  const chosen = [
    ...new Set([
      ...missing.filter((m) => !off.has(m.id ?? '')).map((m) => m.folder ?? ''),
      ...different.filter((m) => !off.has(m.id)).map((m) => m.folder),
    ]),
  ]
  const toggle = (id: string) =>
    setOff((s) => {
      const next = new Set(s)
      if (!next.delete(id)) {
        next.add(id)
      }
      return next
    })
  const sync = (move: boolean) => {
    setMoving(move)
    return run(
      async () => {
        const res = await SyncGameModsFolders(game, profile.id, chosen, move)
        await Promise.all([useProfiles.getState().load(game), useMods.getState().load()])
        setResult(res)
        useToasts.getState().push({
          kind: res.failed > 0 ? 'warning' : 'success',
          title: plural(res.imported, {
            one: move ? 'Moved # mod' : 'Copied # mod',
            other: move ? 'Moved # mods' : 'Copied # mods',
          }),
          changes: [res.profile.lastChange ?? ''],
        })
      },
      { errorTitle: t`Could not change the mods` },
    )
  }
  const detail = result === null ? '' : formatOutcomeDetail(result.outcomes ?? [])
  return (
    <Dialog
      open={true}
      onClose={busy ? undefined : onClose}
      slotProps={{ paper: { sx: { minWidth: 420, maxWidth: 'min(640px, calc(100vw - 64px))' } } }}
    >
      <DialogTitle>{t`Game Mods folder vs ${profile.name}`}</DialogTitle>
      <DialogContent>
        {busy ? <GameModsProgressLine progress={progress} moving={moving} /> : null}
        {result !== null && !busy ? (
          <Section title={t`Done`}>
            <Typography sx={{ fontSize: 14 }}>
              {[
                plural(result.imported, { one: '# imported', other: '# imported' }),
                plural(result.skipped, { one: '# skipped', other: '# skipped' }),
                plural(result.failed, { one: '# failed', other: '# failed' }),
              ].join(' · ')}
            </Typography>
            {detail === '' ? null : (
              <Typography
                sx={{ fontSize: 14, color: 'text.secondary', whiteSpace: 'pre-line', mt: 0.5 }}
              >
                {detail}
              </Typography>
            )}
          </Section>
        ) : null}
        {busy || result !== null ? null : <ReviewLists diff={diff} off={off} onToggle={toggle} />}
      </DialogContent>
      <DialogActions>
        {result === null ? (
          <>
            <Button onClick={onClose} disabled={busy}>
              {t`Cancel`}
            </Button>
            <LockedReason locked={locked}>
              <Button
                variant="outlined"
                disabled={busy || locked || chosen.length === 0}
                onClick={() => sync(true)}
              >
                {t`Move into profile`}
              </Button>
            </LockedReason>
            <LockedReason locked={locked}>
              <Button
                variant="contained"
                disabled={busy || locked || chosen.length === 0}
                onClick={() => sync(false)}
              >
                {t`Copy into profile`}
              </Button>
            </LockedReason>
          </>
        ) : (
          <Button variant="contained" disabled={busy} onClick={onClose}>
            {t`Close`}
          </Button>
        )}
      </DialogActions>
    </Dialog>
  )
}
