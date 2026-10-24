import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Chip, ListItem } from '@mui/material'
import { ArrowRight } from 'lucide-react'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { sourceLabel } from '../../brand/sources/sourceLabel.ts'
import { formatKb } from '../../i18n/bytes.ts'
import { listNames } from '../../i18n/list.ts'
import { useProfiles } from '../../profiles/store.ts'
import { useQueue } from '../../queue/store.ts'
import { Tooltip } from '../../shell/Tooltip.tsx'
import { space } from '../../theme/density.ts'
import { changelogsBetween, changelogsHaveRiskyNotes } from '../changelogRange.ts'
import { sameId, siblingsOf } from '../lookup.ts'
import { openPage } from '../menu.ts'
import { nexusEntryOf, useNexusDetails } from '../nexusDetails.ts'
import { LetterTile } from '../parts.tsx'
import { useMods } from '../store.ts'
import { CautionAck } from './CautionAck.tsx'
import { ROW_TILE } from './constants.ts'
import { dependencyChanges, rowDetails } from './details.ts'
import { OptionalUpdates } from './OptionalUpdates.tsx'
import { RowCopy } from './RowCopy.tsx'
import { UpdateActions } from './UpdateActions.tsx'
import { Version } from './Version.tsx'
import { pendingUpdate } from './wants.ts'

function useRowNotes(update: Update, profileId: string) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const mod = mods.find((m) => m.key === update.key && sameId(m.id, update.id))
  const entries = useProfiles((s) => s.profiles.find((p) => p.id === profileId)?.entries)
  const entry = entries?.find((e) => e.key === update.key)
  const optional = entries?.filter((e) => e.overlayOf === update.key).length ?? 0
  const notes = [
    ...(mod
      ? siblingsOf(mods, mod).map((o) => t`Also updates ${{ name: o.name }} (same download)`)
      : []),
    ...(mod && !mod.enabled ? [t`Disabled in this profile`] : []),
    ...(entry?.pinned && entry.pinReason ? [t`Pinned: ${{ pinned: entry.pinReason }}`] : []),
    ...(optional > 0
      ? [
          plural(
            { count: optional },
            {
              one: '# optional file will be re-applied; check it still fits this version',
              other: '# optional files will be re-applied; check they still fit this version',
            },
          ),
        ]
      : []),
    ...(update.pickFile
      ? [
          t`Pick the file on ${{ source: sourceLabel('nexus') }}: its page has several downloads and none replaces the one you have.`,
        ]
      : []),
    ...(update.unofficial ? [t`Unofficial`] : []),
    ...(update.notDistributable
      ? [
          t`${{ source: update.source }} does not allow downloads outside its own page: get it there.`,
        ]
      : []),
    ...(update.switch && !update.notDistributable
      ? [
          t`From ${{ source: update.source }}, not the site you installed it from. Updating switches its source.`,
        ]
      : []),
  ]
  return { mod, entry, notes }
}

function VersionLine({ update }: { update: Update }) {
  const { t } = useLingui()
  const { added, removed } = dependencyChanges(update)
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, minWidth: 0 }}>
      <Version>{update.installed}</Version>
      <ArrowRight size={14} aria-hidden={true} />
      <Version isNew={true}>{update.version}</Version>
      {added.length + removed.length > 0 ? (
        <Tooltip
          describeChild={true}
          title={
            <>
              {added.length > 0 ? <div>{t`Now needs ${{ names: listNames(added) }}`}</div> : null}
              {removed.length > 0 ? (
                <div>{t`No longer needs ${{ names: listNames(removed) }}`}</div>
              ) : null}
            </>
          }
        >
          <Chip size="small" variant="outlined" tabIndex={0} label={t`Changes dependencies`} />
        </Tooltip>
      ) : null}
    </Box>
  )
}

function ChangeSummary({
  update,
  sizeKb,
  changelog,
}: {
  update: Update
  sizeKb: number
  changelog: string
}) {
  const { t } = useLingui()
  return sizeKb > 0 || changelog ? (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, minWidth: 0, fontSize: 12 }}>
      {sizeKb > 0 ? (
        <Box component="span" sx={{ color: 'text.secondary', flexShrink: 0 }}>
          {formatKb(sizeKb)}
        </Box>
      ) : null}
      {changelog ? (
        <>
          <Tooltip title={changelog}>
            <Box
              component="span"
              tabIndex={0}
              sx={{
                color: 'text.secondary',
                minWidth: 0,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                whiteSpace: 'nowrap',
              }}
            >
              {changelog}
            </Box>
          </Tooltip>
          {update.url ? (
            <Button size="small" sx={{ flexShrink: 0 }} onClick={() => openPage(update.url)}>
              {t`Changelog`}
            </Button>
          ) : null}
        </>
      ) : null}
    </Box>
  ) : null
}

function Row({
  update,
  profileId,
  caution,
  acked,
  onAck,
  picture,
}: {
  update: Update
  profileId: string
  caution: string
  acked: boolean
  onAck: (on: boolean) => void
  picture?: string
}) {
  const { mod, entry, notes } = useRowNotes(update, profileId)
  const game = useProfiles((s) => s.game?.id ?? '')
  const details = useNexusDetails((s) => nexusEntryOf(s.byKey, game, update.nexusId)?.details)
  const riskyChangelog =
    update.nexusId > 0 && details
      ? changelogsHaveRiskyNotes(
          changelogsBetween(details.changelogs ?? [], update.installed, update.version),
        )
      : false
  const { sizeKb, changelog } = rowDetails(update, details)
  const queued = useQueue((s) => pendingUpdate(s.state.items, profileId, update))
  const reportedElsewhere =
    entry?.source.kind === 'nexus' && update.source !== '' && update.source !== 'Nexus'
  return (
    <ListItem
      disablePadding={true}
      sx={{
        display: 'grid',
        gridTemplateColumns: `${ROW_TILE}px minmax(0, 1fr) auto`,
        gap: '14px',
        alignItems: 'center',
        px: space.pad,
        py: space.pad,
        borderBottom: '1px solid var(--mortar-hairline-muted)',
      }}
    >
      <LetterTile
        mod={{ id: update.id, name: update.name, ...(picture ? { picture } : {}) }}
        size={ROW_TILE}
      />
      <Box sx={{ minWidth: 0, display: 'flex', flexDirection: 'column', gap: 0.75 }}>
        <RowCopy
          update={update}
          caution={caution}
          notes={notes}
          reportedElsewhere={reportedElsewhere}
          riskyChangelog={riskyChangelog}
        />
        {caution ? <CautionAck acked={acked} onAck={onAck} /> : null}
        <VersionLine update={update} />
        <ChangeSummary update={update} sizeKb={sizeKb} changelog={changelog} />
        <OptionalUpdates update={update} profileId={profileId} />
      </Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
        <UpdateActions
          update={update}
          {...(mod ? { mod } : {})}
          {...(entry ? { entry } : {})}
          queued={queued}
          caution={caution}
          acked={acked}
        />
      </Box>
    </ListItem>
  )
}

export { Row }
