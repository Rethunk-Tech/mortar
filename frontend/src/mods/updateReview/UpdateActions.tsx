import { useLingui } from '@lingui/react/macro'
import { Button, IconButton, Menu } from '@mui/material'
import { usePopoverAnchor } from '@rethunk/react-load-hooks'
import { Ban, BellOff, ExternalLink, MoreHorizontal, Pin, PinOff, RefreshCcw } from 'lucide-react'
import { useState } from 'react'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type {
  Mod,
  Profile,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { openProfileOf, useProfiles } from '../../profiles/store.ts'
import { download } from '../../queue/actions.ts'
import { DisabledReason } from '../../shell/DisabledReason.tsx'
import { ConfirmationDialog } from '../../shell/dialog/ConfirmationDialog.tsx'
import { MenuAction } from '../../shell/MenuAction.tsx'
import { OfflineGate } from '../../shell/OfflineGate.tsx'
import { updateSources, useOfflineReason } from '../../shell/offlineText.ts'
import { StableLabel } from '../../shell/StableLabel.tsx'
import { reportUnexpected } from '../../toasts/report.ts'
import { installableUpdate } from '../lookup.ts'
import { ICON_SIZE, openPage } from '../menu.ts'
import { detailsFor, useNexusDetails } from '../nexusDetails.ts'
import { useOptionalSkips } from '../optionalFiles.ts'
import { useMods } from '../store.ts'
import { useEverywhere } from '../useEverywhere.ts'
import { EverywhereDialog } from './EverywhereDialog.tsx'
import { withOptional } from './wants.ts'

export function UpdateActions({
  update,
  mod,
  entry,
  queued,
  caution,
  acked,
}: {
  update: Update
  mod?: Mod
  entry?: NonNullable<Profile['entries']>[number]
  queued: boolean
  caution: string
  acked: boolean
}) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const setPinned = useMods((s) => s.setPinned)
  const setSkipVersion = useMods((s) => s.setSkipVersion)
  const setSkipSource = useMods((s) => s.setSkipSource)
  const { anchor, show, hide } = usePopoverAnchor()
  const [everywhere, setEverywhere] = useState(false)
  const [confirmSwitch, setConfirmSwitch] = useState(false)
  const offline = useOfflineReason(updateSources(update))
  const { count: n, why: everywhereWhy } = useEverywhere(game, update.id, update)
  const everywhereBlocked = everywhereWhy !== ''
  const blocked = queued || (caution !== '' && !acked)
  let everywhereReason = everywhereWhy
  if (!everywhereBlocked) {
    everywhereReason = queued
      ? t`This update is already queued.`
      : t`Tick "Update despite this" first.`
  }
  const missingWhy = t`This mod is not in the profile.`
  const act = (fn: () => Promise<unknown>) => () => {
    hide()
    fn().catch(reportUnexpected)
  }
  const install = () => {
    const byId = detailsFor(useNexusDetails.getState().byKey, game)
    const skipped = useOptionalSkips.getState().skipped[update.key] === true
    const profile = openProfileOf(useProfiles.getState())
    download(
      withOptional(update, profile, byId[update.nexusId]?.details?.files ?? [], skipped),
    ).catch(reportUnexpected)
  }
  return (
    <>
      {installableUpdate(update) ? (
        <OfflineGate reason={offline}>
          <Button
            variant="contained"
            disabled={blocked}
            onClick={update.switch ? () => setConfirmSwitch(true) : install}
          >
            <StableLabel labels={[t`Update`, t`Queued`]} shown={queued ? t`Queued` : t`Update`} />
          </Button>
        </OfflineGate>
      ) : null}
      <IconButton
        aria-label={t`More actions for ${{ name: update.name }}`}
        aria-haspopup="menu"
        aria-expanded={anchor !== null}
        onClick={show}
        sx={{ borderRadius: '6px', bgcolor: anchor ? 'var(--mortar-hairline)' : 'transparent' }}
      >
        <MoreHorizontal size={18} />
      </IconButton>
      <Menu open={anchor !== null} anchorEl={anchor} onClose={() => hide()}>
        {update.url ? (
          <MenuAction
            icon={<ExternalLink size={ICON_SIZE} />}
            label={t`Open page`}
            onClick={() => {
              hide()
              openPage(update.url)
            }}
          />
        ) : null}
        <DisabledReason title={missingWhy} disabled={!mod}>
          <MenuAction
            icon={entry?.pinned ? <PinOff size={ICON_SIZE} /> : <Pin size={ICON_SIZE} />}
            label={entry?.pinned ? t`Unpin version` : t`Pin this version`}
            disabled={!mod}
            onClick={act(async () => mod && setPinned(mod, !entry?.pinned))}
          />
        </DisabledReason>
        <DisabledReason title={missingWhy} disabled={!mod}>
          <MenuAction
            icon={<Ban size={ICON_SIZE} />}
            label={t`Skip this update`}
            disabled={!mod}
            onClick={act(async () => mod && setSkipVersion(mod, update.version))}
          />
        </DisabledReason>
        {installableUpdate(update) ? (
          <DisabledReason title={everywhereReason} disabled={everywhereBlocked || blocked}>
            <MenuAction
              icon={<RefreshCcw size={ICON_SIZE} />}
              label={t`Update in all profiles (${{ count: n }})`}
              disabled={everywhereBlocked || blocked}
              onClick={() => {
                hide()
                setEverywhere(true)
              }}
            />
          </DisabledReason>
        ) : null}
        {update.source && !entry?.skipSources?.includes(update.source) ? (
          <DisabledReason title={missingWhy} disabled={!mod}>
            <MenuAction
              icon={<BellOff size={ICON_SIZE} />}
              label={t`Ignore updates from ${{ source: update.source }}`}
              disabled={!mod}
              onClick={act(async () => mod && setSkipSource(mod, update.source, true))}
            />
          </DisabledReason>
        ) : null}
      </Menu>
      <ConfirmationDialog
        open={confirmSwitch}
        title={t`Switch ${{ name: update.name }} to ${{ source: update.source }}?`}
        body={t`${{ name: update.name }} was installed from another site. Updating installs ${{ version: update.version }} from ${{ source: update.source }} instead and keeps your settings.`}
        confirmLabel={t`Switch and update`}
        onCancel={() => setConfirmSwitch(false)}
        onConfirm={() => {
          setConfirmSwitch(false)
          install()
        }}
      />
      <EverywhereDialog
        open={everywhere}
        game={game}
        mods={[{ id: update.id, newKey: 'latest' }]}
        onClose={() => setEverywhere(false)}
      />
    </>
  )
}
