import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from '@mui/material'
import { useTheme } from '@mui/material/styles'
import { CircleAlert } from 'lucide-react'
import { useEffect } from 'react'
import { Hint } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'
import { STEAM } from '../brand/productNames.ts'
import { CrashDialog, SwitchOffButton } from '../console/CrashDialog.tsx'
import { useTab } from '../game/tab.ts'
import { useGameInfo, useGameName } from '../games/info.ts'
import { useReasonText } from '../i18n/reasonText.ts'
import { nexusIdOf } from '../mods/lookup.ts'
import { openPage } from '../mods/menu.ts'
import { openPageLabel } from '../mods/modActions.ts'
import { nexusDomain } from '../mods/nexusDomain.ts'
import { nexusModUrl } from '../mods/nexusUrl.ts'
import { useMods } from '../mods/store.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { FlatpakGrant } from '../shell/FlatpakGrant.tsx'
import { SyncOffers } from '../sync/SyncOffers.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { GraphicsDialog } from './GraphicsDialog.tsx'
import { KnownGoodOffer } from './KnownGoodOffer.tsx'
import { LaunchScreen } from './LaunchScreen.tsx'
import { PrePlayDialog } from './PrePlayDialog.tsx'
import { cancelling } from './playModeState.ts'
import { useLaunchView } from './progress.ts'
import { SaveWarnDialog } from './SaveWarnDialog.tsx'
import { useLaunch } from './store.ts'
import { UpdateWarnDialog } from './UpdateWarnDialog.tsx'
import { VanillaPlayDialogs } from './VanillaPlay.tsx'

function Failure({ game }: { game: string }) {
  const { t, i18n } = useLingui()
  const reasonText = useReasonText()
  const theme = useTheme()
  const info = useProfiles((s) => s.game)
  const failure = useLaunch((s) => s.failure)
  const dismiss = useLaunch((s) => s.dismissFailure)
  const close = cancelling(dismiss)
  const start = useLaunch((s) => s.start)
  const cause = failure?.cause
  const mod = useMods((s) => s.mods.find((m) => m.key === cause?.modKey))
  const profile = useProfiles(openProfileOf)
  // A launch screen that is up shows the failure itself; one that never drew (a launch that failed before it began)
  // leaves it to this dialog.
  const onScreen = useLaunchView(
    (s) => s.phase === 'failed' && s.game === game && s.progress !== null,
  )
  const hidden = useLaunch((s) => s.hidden)
  // The dialog unmounts with `failure`, so nothing fades out with stale text.
  if (!failure || (onScreen && !hidden)) {
    return null
  }
  const name = info?.name ?? ''
  const nexusID = profile && mod ? nexusIdOf(profile, mod) : 0
  const showFlatpak = failure.hint === Hint.HintFlatpakFS
  return (
    <Dialog
      open={true}
      onClose={close}
      fullWidth={true}
      maxWidth="sm"
      slotProps={{ paper: { sx: { bgcolor: 'var(--mortar-panel-solid)' } } }}
    >
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
        <CircleAlert size={22} color={theme.palette.error.light} aria-hidden={true} />
        {t`${name} did not start`}
      </DialogTitle>
      <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: space.pad }}>
        <DialogContentText sx={{ whiteSpace: 'pre-wrap', userSelect: 'text' }}>
          {failure.body || t`The game exited before it started. The console has its output.`}
        </DialogContentText>
        {cause ? (
          <DialogContentText>
            <strong>{t`Caused by ${{ name: cause.modName }}`}</strong>
            <br />
            {reasonText(cause.reason, cause.params, cause.detail)}
          </DialogContentText>
        ) : null}
        {showFlatpak ? <FlatpakGrant /> : null}
      </DialogContent>
      <DialogActions>
        <Button onClick={close}>{t`Close`}</Button>
        <Button
          onClick={() => {
            dismiss()
            useTab.getState().setTab('console')
          }}
        >
          {t`Open console`}
        </Button>
        {mod ? <SwitchOffButton mod={mod} /> : null}
        {nexusID > 0 ? (
          <Button onClick={() => openPage(nexusModUrl(nexusID, nexusDomain()))}>
            {openPageLabel(i18n, 'nexus')}
          </Button>
        ) : null}
        <Button
          variant="contained"
          onClick={() => {
            dismiss()
            start(game, failure.profile, false)
          }}
        >
          {t`Retry`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

function DirectDialog() {
  const { t } = useLingui()
  const gameName = useGameName()
  const loader = useGameInfo()?.loader ?? ''
  const ask = useLaunch((s) => s.askDirect)
  const answer = useLaunch((s) => s.answerDirect)
  return (
    <ConfirmationDialog
      open={ask !== null}
      title={t`${{ launcher: STEAM }} was not found`}
      body={
        ask?.profile === ''
          ? t`Mortar can start ${{ name: gameName }} directly instead, without the ${{ launcher: STEAM }} overlay or ${{ launcher: STEAM }}'s playtime tracking.`
          : t`Mortar can start ${{ name: loader }} directly instead, without the ${{ launcher: STEAM }} overlay or ${{ launcher: STEAM }}'s playtime tracking.`
      }
      confirmLabel={t`Launch without ${{ launcher: STEAM }}`}
      onCancel={cancelling(() => {
        answer(false).catch(reportUnexpected)
      })}
      onConfirm={() => {
        answer(true).catch(reportUnexpected)
      }}
    />
  )
}

export function LaunchLayer({ game }: { game: string }) {
  const refresh = useLaunch((s) => s.refresh)
  useEffect(() => {
    if (game) {
      refresh(game)
    }
  }, [game, refresh])
  return (
    <>
      <LaunchScreen game={game} />
      <Failure game={game} />
      <DirectDialog />
      <CrashDialog />
      <UpdateWarnDialog />
      <SaveWarnDialog />
      <GraphicsDialog />
      <PrePlayDialog />
      <KnownGoodOffer />
      <SyncOffers />
      <VanillaPlayDialogs />
    </>
  )
}
