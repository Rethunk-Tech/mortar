import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Menu,
  Tooltip,
  Typography,
} from '@mui/material'
import { ChevronDown, ExternalLink, LifeBuoy, Search, Terminal } from 'lucide-react'
import { useState } from 'react'
import { Start as StartBisect } from '../../bindings/github.com/Rethunk-AI/mortar/internal/bisect/service.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { useTab } from '../game/tab.ts'
import { useLaunch } from '../launch/store.ts'
import { nexusIdOf } from '../mods/lookup.ts'
import { openPage } from '../mods/menu.ts'
import { paper } from '../mods/paper.ts'
import { ReportToAuthorButton } from '../mods/ReportToAuthorButton.tsx'
import { useMods } from '../mods/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { errorDetails } from '../toasts/errorKind.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { BisectDialog } from './BisectDialog.tsx'
import { canBisectCrash } from './canBisect.ts'
import { crashCauseDetailLine, crashCauseKind } from './crashCause.ts'
import { useConsole } from './store.ts'

type Crash = NonNullable<ReturnType<typeof useLaunch.getState>['crash']>

// Switching the suspected mod off, with an Undo toast; busy while the profile write runs.
function SwitchOffButton({ mod, primary = false }: { mod: Mod; primary?: boolean }) {
  const { t } = useLingui()
  const [switching, setSwitching] = useState(false)
  return (
    <Button
      variant={primary ? 'contained' : 'text'}
      disabled={switching}
      onClick={() => {
        setSwitching(true)
        useMods
          .getState()
          .setEnabled(mod, false)
          .then(() => {
            useToasts.getState().push({
              kind: 'success',
              title: t`Switched off ${mod.name}`,
              action: {
                label: t`Undo`,
                run: () => useMods.getState().setEnabled(mod, true),
              },
            })
          })
          .catch(reportUnexpected)
          .finally(() => setSwitching(false))
      }}
    >
      {t`Switch off`}
    </Button>
  )
}

function openRun(crash: Crash, console: boolean) {
  useProfiles.getState().open(crash.profile)
  if (console) {
    useTab.getState().setTab('console')
  }
  useConsole.getState().viewRun(crash.game, crash.profile, crash.runId)
}

// Everything but the one primary action, so the row never wraps into a wall of buttons.
function MoreActions({
  crash,
  nexusID,
  onBisect,
  withConsole,
  onDone,
}: {
  crash: Crash
  nexusID: number
  onBisect: (() => void) | null
  withConsole: boolean
  onDone: () => void
}) {
  const { t } = useLingui()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const close = () => setAnchor(null)
  return (
    <>
      <Button endIcon={<ChevronDown size={16} />} onClick={(e) => setAnchor(e.currentTarget)}>
        {t`More`}
      </Button>
      <Menu anchorEl={anchor} open={anchor !== null} onClose={close}>
        {onBisect ? (
          <MenuAction
            icon={<Search size={16} />}
            label={t`Find the mod causing this`}
            onClick={() => {
              close()
              onBisect()
            }}
          />
        ) : null}
        {nexusID > 0 ? (
          <MenuAction
            icon={<ExternalLink size={16} />}
            label={t`Open on Nexus`}
            onClick={() => {
              close()
              openPage(`https://www.nexusmods.com/stardewvalley/mods/${nexusID}`).catch(
                reportUnexpected,
              )
            }}
          />
        ) : null}
        <MenuAction
          icon={<LifeBuoy size={16} />}
          label={t`Share log…`}
          onClick={() => {
            close()
            openRun(crash, false)
            useConsole.getState().setHelping(true)
            onDone()
          }}
        />
        {withConsole ? (
          <MenuAction
            icon={<Terminal size={16} />}
            label={t`Open console`}
            onClick={() => {
              close()
              openRun(crash, true)
              onDone()
            }}
          />
        ) : null}
      </Menu>
    </>
  )
}

export function CrashDialog() {
  const { t } = useLingui()
  const crash = useLaunch((s) => s.crash)
  const dismiss = useLaunch((s) => s.dismissCrash)
  const [bisectJob, setBisectJob] = useState<{ id: string; game: string; profile: string } | null>(
    null,
  )
  const [bisectError, setBisectError] = useState<string | null>(null)
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === crash?.profile))
  const mod = useMods((s) => s.mods.find((m) => m.key === crash?.cause?.modKey))
  if (!crash) {
    return bisectJob ? (
      <BisectDialog
        game={bisectJob.game}
        profile={bisectJob.profile}
        jobID={bisectJob.id}
        onClose={() => setBisectJob(null)}
      />
    ) : null
  }
  const canBisect = canBisectCrash(crash)
  const nexusID = profile && mod ? nexusIdOf(profile, mod) : 0
  const startBisect = async () => {
    try {
      const id = await StartBisect(crash.game, crash.profile)
      setBisectJob({ id, game: crash.game, profile: crash.profile })
      setBisectError(null)
      dismiss()
    } catch (error) {
      setBisectError(errorDetails(error))
    }
  }
  const causeKind = crash.cause ? crashCauseKind(crash.cause.reason) : ''
  let causeBody = crash.cause ? crashCauseDetailLine(crash.cause.detail) : ''
  if (causeKind === 'missing-file') {
    causeBody = t`A file it needs could not be opened.`
  } else if (causeKind === 'asset-load') {
    causeBody = t`It could not load an asset.`
  } else if (causeKind === 'mod-exception') {
    causeBody = t`It encountered an error.`
  }
  return (
    <>
      <Dialog
        open={true}
        onClose={dismiss}
        transitionDuration={0}
        slotProps={{ paper: { sx: { ...paper.sx, width: 520, maxWidth: 'calc(100% - 32px)' } } }}
      >
        <DialogTitle sx={{ fontSize: 22, fontWeight: 700 }}>
          {crash.crashed ? t`Stardew Valley crashed` : t`Stardew Valley closed with errors`}
        </DialogTitle>
        <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
          {crash.cause ? (
            <Box sx={{ fontSize: 14 }}>
              <Box sx={{ fontWeight: 700 }}>{t`Caused by ${crash.cause.modName}`}</Box>
              <Box sx={{ color: 'text.secondary', mt: 0.25 }} title={crash.cause.detail}>
                {causeBody}
              </Box>
            </Box>
          ) : null}
          {crash.mods === null || crash.mods.length === 0 ? (
            <Tooltip title={t`SMAPI`}>
              <Typography sx={{ fontSize: 14 }}>{t`The game log has errors.`}</Typography>
            </Tooltip>
          ) : (
            crash.mods.map((row: { mod: string; count: number; first: string }) => (
              <Box key={row.mod} sx={{ fontSize: 14, lineHeight: 1.45 }}>
                <Box sx={{ fontWeight: 700 }}>
                  {t`${row.mod} · ${plural(row.count, { one: '# error', other: '# errors' })}`}
                </Box>
                <Box sx={{ color: 'text.secondary', mt: 0.25 }}>{row.first}</Box>
              </Box>
            ))
          )}
          {bisectError ? (
            <Typography color="error" title={errorDetails(bisectError)}>
              {errorMessage(bisectError)}
            </Typography>
          ) : null}
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2.5, gap: 1 }}>
          <Button onClick={dismiss} sx={{ whiteSpace: 'nowrap', mr: 'auto' }}>
            {t`Dismiss`}
          </Button>
          <MoreActions
            crash={crash}
            nexusID={nexusID}
            onBisect={canBisect ? startBisect : null}
            withConsole={Boolean(mod)}
            onDone={dismiss}
          />
          {mod && profile && crash.cause ? (
            <ReportToAuthorButton
              game={crash.game}
              profile={profile}
              runId={crash.runId}
              mod={mod}
              modLogName={crash.cause.modName}
            />
          ) : null}
          {mod ? (
            <SwitchOffButton mod={mod} primary={true} />
          ) : (
            <Button
              variant="contained"
              startIcon={<Terminal size={16} />}
              onClick={() => {
                openRun(crash, true)
                dismiss()
              }}
              sx={{ whiteSpace: 'nowrap' }}
            >
              {t`Open console`}
            </Button>
          )}
        </DialogActions>
      </Dialog>
      {bisectJob ? (
        <BisectDialog
          game={bisectJob.game}
          profile={bisectJob.profile}
          jobID={bisectJob.id}
          onClose={() => setBisectJob(null)}
        />
      ) : null}
    </>
  )
}

export { SwitchOffButton }
