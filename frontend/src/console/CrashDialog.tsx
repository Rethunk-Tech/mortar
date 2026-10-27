import { useLingui } from '@lingui/react/macro'
import { Box, Button, Link, Menu, Typography } from '@mui/material'
import { ChevronDown, ExternalLink, LifeBuoy, Search, Terminal } from 'lucide-react'
import { useState } from 'react'
import { Start as StartBisect } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/bisect/service.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useTab } from '../game/tab.ts'
import { useGameName } from '../games/info.ts'
import { errorsLabel } from '../i18n/counts.ts'
import { i18n } from '../i18n/index.ts'
import { useReasonText } from '../i18n/reasonText.ts'
import { useLaunch } from '../launch/store.ts'
import { nexusIdOf } from '../mods/lookup.ts'
import { openPage } from '../mods/menu.ts'
import { openPageLabel } from '../mods/modActions.ts'
import { nexusDomain } from '../mods/nexusDomain.ts'
import { nexusModUrl } from '../mods/nexusUrl.ts'
import { openMod } from '../mods/openMod.ts'
import { ReportToAuthorButton } from '../mods/ReportToAuthorButton.tsx'
import { useMods } from '../mods/store.ts'
import { useProfileLoader, useProfiles } from '../profiles/store.ts'
import { AppDialog, DialogFooter } from '../shell/dialog/AppDialog.tsx'
import { ErrorRetry } from '../shell/ErrorRetry.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { space } from '../theme/density.ts'
import { errorDetails } from '../toasts/errorKind.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { BisectConfirm } from './BisectConfirm.tsx'
import { BisectDialog } from './BisectDialog.tsx'
import { crashCauseDetailLine } from './crashCause.ts'
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
              title: t`Disabled ${{ names: mod.name }}`,
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
      {t`Disable`}
    </Button>
  )
}

function showMod(name: string, onDone: () => void) {
  const mod = useMods.getState().mods.find((m) => m.name === name)
  if (!mod) {
    return
  }
  openMod({ id: mod.id, key: mod.key }).catch(reportUnexpected)
  onDone()
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
  withConsole,
  onDone,
}: {
  crash: Crash
  nexusID: number
  withConsole: boolean
  onDone: () => void
}) {
  const { t } = useLingui()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const profileLoader = useProfileLoader(crash.profile)
  const shares = profileLoader?.share === true || Boolean(profileLoader?.paste)
  const close = () => setAnchor(null)
  return (
    <>
      <Button
        endIcon={<ChevronDown size={16} />}
        aria-haspopup="menu"
        aria-expanded={anchor !== null}
        onClick={(e) => setAnchor(e.currentTarget)}
      >
        {t`More`}
      </Button>
      <Menu anchorEl={anchor} open={anchor !== null} onClose={close}>
        {nexusID > 0 ? (
          <MenuAction
            icon={<ExternalLink size={16} />}
            label={openPageLabel(i18n, 'nexus')}
            onClick={() => {
              close()
              openPage(nexusModUrl(nexusID, nexusDomain()))
            }}
          />
        ) : null}
        {shares ? (
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
        ) : null}
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

// The one main action: disable the mod that caused it, find the cause by halving, or read the log.
function CrashPrimary({
  mod,
  canBisect,
  onBisect,
  onConsole,
}: {
  mod: Mod | undefined
  canBisect: boolean
  onBisect: () => void
  onConsole: () => void
}) {
  const { t } = useLingui()
  if (mod) {
    return <SwitchOffButton mod={mod} primary={true} />
  }
  if (canBisect) {
    return (
      <Button variant="contained" startIcon={<Search size={16} />} onClick={onBisect}>
        {t`Find the mod that crashes the game…`}
      </Button>
    )
  }
  return (
    <Button variant="contained" startIcon={<Terminal size={16} />} onClick={onConsole}>
      {t`Open console`}
    </Button>
  )
}

export function CrashDialog() {
  const { t } = useLingui()
  const crash = useLaunch((s) => s.crash)
  const gameName = useGameName(crash?.game)
  const dismiss = useLaunch((s) => s.dismissCrash)
  const [bisectJob, setBisectJob] = useState<{ id: string; game: string; profile: string } | null>(
    null,
  )
  const [bisectError, setBisectError] = useState<string | null>(null)
  const [asking, setAsking] = useState(false)
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === crash?.profile))
  const mod = useMods((s) => s.mods.find((m) => m.key === crash?.cause?.modKey))
  const reasonText = useReasonText()
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
  const canBisect = crash.cause === null || crash.cause === undefined
  const nexusID = profile && mod ? nexusIdOf(profile, mod) : 0
  const startBisect = async () => {
    try {
      const id = await StartBisect(crash.game, crash.profile)
      setBisectJob({ id, game: crash.game, profile: crash.profile })
      setBisectError(null)
      setAsking(false)
      dismiss()
    } catch (error) {
      setBisectError(errorDetails(error))
      setAsking(false)
    }
  }
  const causeBody = crash.cause
    ? reasonText(crash.cause.reason, crash.cause.params, crashCauseDetailLine(crash.cause.detail))
    : ''
  return (
    <>
      <AppDialog
        open={true}
        onClose={dismiss}
        size="md"
        title={
          crash.crashed
            ? t`${{ game: gameName }} crashed`
            : t`${{ game: gameName }} closed with errors`
        }
        footer={
          <DialogFooter>
            <Button onClick={dismiss}>{t`Dismiss`}</Button>
            <MoreActions
              crash={crash}
              nexusID={nexusID}
              withConsole={Boolean(mod) || canBisect}
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
            <CrashPrimary
              mod={mod}
              canBisect={canBisect}
              onBisect={() => setAsking(true)}
              onConsole={() => {
                openRun(crash, true)
                dismiss()
              }}
            />
          </DialogFooter>
        }
      >
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
          {crash.cause ? (
            <Box sx={{ fontSize: 'medium.fontSize' }}>
              <Box sx={{ fontWeight: 700 }}>{t`Caused by ${{ name: crash.cause.modName }}`}</Box>
              <Box sx={{ color: 'text.secondary', mt: 0.25 }} title={crash.cause.detail}>
                {causeBody}
              </Box>
            </Box>
          ) : null}
          {crash.mods === null || crash.mods.length === 0 ? (
            <Typography sx={{ fontSize: 'medium.fontSize' }}>
              {canBisect
                ? t`Mortar could not tell which mod caused this. Finding the mod tries halves of your mods until the crash stops; your profile is not changed.`
                : t`The game log has errors.`}
            </Typography>
          ) : (
            crash.mods.map((row: { mod: string; count: number; first: string }) => (
              <Box key={row.mod} sx={{ fontSize: 'medium.fontSize', lineHeight: 1.45 }}>
                <Box sx={{ fontWeight: 700 }}>
                  <Link component="button" type="button" onClick={() => showMod(row.mod, dismiss)}>
                    {row.mod}
                  </Link>
                  {` · ${errorsLabel(row.count)}`}
                </Box>
                <Box sx={{ color: 'text.secondary', mt: 0.25 }}>{row.first}</Box>
              </Box>
            ))
          )}
          {bisectError ? (
            <ErrorRetry
              error={{ message: errorMessage(bisectError), details: errorDetails(bisectError) }}
              onRetry={() => {
                startBisect().catch(reportUnexpected)
              }}
            />
          ) : null}
        </Box>
      </AppDialog>
      <BisectConfirm open={asking} onCancel={() => setAsking(false)} onConfirm={startBisect} />
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
