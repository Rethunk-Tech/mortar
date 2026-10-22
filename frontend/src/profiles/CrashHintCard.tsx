import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { FolderOpen, PowerOff, Search, TriangleAlert, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Start as StartBisect } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/bisect/service.ts'
import type { CrashHint as Hint } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/models.ts'
import {
  CrashHint,
  OpenCrashReport,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { BisectDialog } from '../console/BisectDialog.tsx'
import { formatWhen } from '../i18n/formatWhen.ts'
import { useReasonText } from '../i18n/reasonText.ts'
import { useLaunch } from '../launch/store.ts'
import { useMods } from '../mods/store.ts'
import { readStored, writeStored } from '../shell/useStoredState.ts'
import { calloutFill, calloutLine } from '../theme/callout.ts'
import { MONO } from '../theme/theme.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useProfiles } from './store.ts'

const DISMISSED_KEY = 'mortar.crashHintDismissed'

const isText = (v: unknown): v is string => typeof v === 'string'

// Shown at the top of the profile after a crashed run: the mod the log blames with a way to switch it off and play
// again, or, when the log blames none, a way to find it by halving the mods.
export function CrashHintCard({
  game,
  profileId,
  crashedAt,
}: {
  game: string
  profileId: string
  crashedAt: string
}) {
  const { t } = useLingui()
  const updated = useProfiles((s) => s.profiles.find((p) => p.id === profileId)?.updated ?? '')
  const mod = useMods((s) => s.mods)
  const [hint, setHint] = useState<Hint | null>(null)
  const [hidden, setHidden] = useState(() => readStored(DISMISSED_KEY, '', isText))
  const [bisectJob, setBisectJob] = useState('')
  const [busy, setBusy] = useState(false)
  // The profile's updated time moves after each run, when a new hint may exist.
  const key = `${game}\0${profileId}\0${updated}`
  useEffect(() => {
    const [g = '', id = ''] = key.split('\0')
    let live = true
    CrashHint(g, id)
      .then((h) => live && setHint(h))
      .catch(() => live && setHint(null))
    return () => {
      live = false
    }
  }, [key])
  const culprit = hint?.modKey ? mod.find((m) => m.key === hint.modKey) : undefined
  const named = t`named in the crash log`
  const reason = useReasonText()(hint?.code, hint?.params, hint?.reason ?? '')
  const bisect = () => StartBisect(game, profileId).then(setBisectJob).catch(reportUnexpected)
  const retry = (target: NonNullable<typeof culprit>) => {
    setBusy(true)
    useMods
      .getState()
      .setEnabled(target, false)
      .then(() => useLaunch.getState().start(game, profileId, false))
      .catch(reportUnexpected)
      .finally(() => setBusy(false))
  }
  return (
    <>
      {hint && hint.runId !== hidden ? (
        <Box
          role="status"
          sx={{
            gridColumn: '1 / -1',
            display: 'flex',
            alignItems: 'center',
            gap: 1.5,
            flexWrap: 'wrap',
            px: 2,
            py: 1.75,
            bgcolor: (theme) => calloutFill('error')(theme),
            border: (theme) => `1px solid ${calloutLine('error')(theme)}`,
            borderRadius: '10px',
          }}
        >
          <Box component="span" sx={{ display: 'flex', color: 'error.main' }}>
            <TriangleAlert size={18} aria-hidden={true} />
          </Box>
          <Box sx={{ flex: 1, minWidth: 220, display: 'flex', flexDirection: 'column', gap: 0.25 }}>
            <Typography sx={{ fontWeight: 700 }}>
              {crashedAt
                ? t`The game crashed ${{ when: formatWhen(crashedAt) }}`
                : t`The game crashed`}
            </Typography>
            <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
              {culprit
                ? t`Most likely cause: ${{ name: hint.modName }} (${{ reason: reason || named }})`
                : reason || t`The log does not name a mod.`}
            </Typography>
            {hint.nativeFrame ? (
              <Typography
                title={hint.nativeFrame}
                sx={{
                  fontSize: 12,
                  color: 'text.secondary',
                  fontFamily: MONO,
                  wordBreak: 'break-all',
                }}
              >
                {t`Crashed in ${{ nativeFrame: hint.nativeFrame }}`}
              </Typography>
            ) : null}
            {culprit && hint.evidence ? (
              <Typography
                title={hint.evidence}
                sx={{
                  fontSize: 12,
                  color: 'text.secondary',
                  fontFamily: MONO,
                  wordBreak: 'break-all',
                }}
              >
                {t`Stack frame: ${{ evidence: hint.evidence }}`}
              </Typography>
            ) : null}
          </Box>
          {culprit ? (
            <Button
              variant="contained"
              disabled={busy}
              startIcon={<PowerOff size={16} />}
              onClick={() => retry(culprit)}
            >
              {t`Disable it and play`}
            </Button>
          ) : null}
          <Button
            variant="outlined"
            color="inherit"
            startIcon={<Search size={16} />}
            onClick={bisect}
          >
            {t`Find the cause…`}
          </Button>
          {hint.reportDir ? (
            <Button
              variant="outlined"
              color="inherit"
              startIcon={<FolderOpen size={16} />}
              onClick={() => OpenCrashReport(game, profileId).catch(reportUnexpected)}
            >
              {t`Open the crash report`}
            </Button>
          ) : null}
          <Button
            color="inherit"
            startIcon={<X size={16} />}
            onClick={() => {
              writeStored(DISMISSED_KEY, hint.runId)
              setHidden(hint.runId)
            }}
          >
            {t`Dismiss`}
          </Button>
        </Box>
      ) : null}
      {bisectJob ? (
        <BisectDialog
          game={game}
          profile={profileId}
          jobID={bisectJob}
          onClose={() => setBisectJob('')}
        />
      ) : null}
    </>
  )
}
