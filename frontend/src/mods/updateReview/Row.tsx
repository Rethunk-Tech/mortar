import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Chip, Tooltip } from '@mui/material'
import { ArrowRight, ExternalLink } from 'lucide-react'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { formatKb } from '../../i18n/bytes.ts'
import { listNames } from '../../i18n/list.ts'
import { useProfiles } from '../../profiles/store.ts'
import { useQueue } from '../../queue/store.ts'
import { changelogsBetween, changelogsHaveRiskyNotes } from '../changelogRange.ts'
import { sameId, siblingsOf } from '../lookup.ts'
import { openPage } from '../menu.ts'
import { useNexusDetails } from '../nexusDetails.ts'
import { LetterTile } from '../parts.tsx'
import { useMods } from '../store.ts'
import { ROW_TILE } from './constants.ts'
import { dependencyChanges, rowDetails } from './details.ts'
import { OptionalUpdates } from './OptionalUpdates.tsx'
import { RowCopy } from './RowCopy.tsx'
import { RowInclude } from './RowInclude.tsx'
import { UpdateActions } from './UpdateActions.tsx'
import { Version } from './Version.tsx'
import { pendingUpdate } from './wants.ts'

export function Row({
  update,
  profileId,
  caution,
  acked,
  included,
  onAck,
  onInclude,
  picture,
}: {
  update: Update
  profileId: string
  caution: string
  acked: boolean
  included: boolean
  onAck: (on: boolean) => void
  onInclude: (on: boolean) => void
  picture?: string
}) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const mod = mods.find((m) => m.key === update.key && sameId(m.id, update.id))
  const entries = useProfiles((s) => s.profiles.find((p) => p.id === profileId)?.entries)
  const entry = entries?.find((e) => e.key === update.key)
  const optional = entries?.filter((e) => e.overlayOf === update.key).length ?? 0
  const details = useNexusDetails((s) => s.byId[update.nexusId]?.details)
  const riskyChangelog =
    update.nexusId > 0 && details
      ? changelogsHaveRiskyNotes(
          changelogsBetween(details.changelogs ?? [], update.installed, update.version),
        )
      : false
  const { sizeKb, changelog } = rowDetails(update, details)
  const { added, removed } = dependencyChanges(update)
  const queued = useQueue((s) => pendingUpdate(s.state.items, profileId, update))
  const notes = [
    ...(mod ? siblingsOf(mods, mod).map((o) => t`Also updates ${o.name} (same download)`) : []),
    ...(mod && !mod.enabled ? [t`Disabled in this profile`] : []),
    ...(entry?.pinned && entry.pinReason ? [t`Pinned: ${entry.pinReason}`] : []),
    ...(optional > 0
      ? [
          t`${plural(optional, { one: '# optional file will be re-applied; check it still fits this version', other: '# optional files will be re-applied; check they still fit this version' })}`,
        ]
      : []),
    ...(update.unofficial ? [t`Unofficial`] : []),
    ...(update.switch
      ? [
          t`From ${update.source}, not the site you installed it from. Updating switches its source.`,
        ]
      : []),
    ...(update.githubFallback && !update.githubRepo
      ? [t`From GitHub (${update.githubFallback}) when its release matches, else Nexus`]
      : []),
  ]
  const reportedElsewhere =
    entry?.source.kind === 'nexus' && update.source !== '' && update.source !== 'Nexus'
  return (
    <Box
      role="listitem"
      sx={{
        display: 'grid',
        gridTemplateColumns: caution
          ? `${ROW_TILE}px minmax(0, 1fr) auto auto auto`
          : `${ROW_TILE}px minmax(0, 1fr) auto auto`,
        gap: '14px',
        alignItems: 'center',
        px: 3,
        py: 1.75,
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
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 0 }}>
          <Version>{update.installed}</Version>
          <ArrowRight size={14} aria-hidden={true} />
          <Version isNew={true}>{update.version}</Version>
          {update.source ? <Chip size="small" variant="outlined" label={update.source} /> : null}
          {added.length + removed.length > 0 ? (
            <Tooltip
              title={
                <>
                  {added.length > 0 ? <div>{t`Now needs ${listNames(added)}`}</div> : null}
                  {removed.length > 0 ? (
                    <div>{t`No longer needs ${listNames(removed)}`}</div>
                  ) : null}
                </>
              }
            >
              <Chip size="small" variant="outlined" label={t`Changes dependencies`} />
            </Tooltip>
          ) : null}
        </Box>
        {sizeKb > 0 || changelog ? (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 0, fontSize: 12 }}>
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
        ) : null}
        <OptionalUpdates update={update} profileId={profileId} />
      </Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        {update.url ? (
          <Button
            variant="outlined"
            endIcon={<ExternalLink size={12} />}
            onClick={() => openPage(update.url)}
          >
            {t`Open page`}
          </Button>
        ) : null}
        <UpdateActions
          update={update}
          {...(mod ? { mod } : {})}
          {...(entry ? { entry } : {})}
          queued={queued}
          caution={caution}
          acked={acked}
        />
      </Box>
      <RowInclude
        name={update.name}
        caution={caution}
        acked={acked}
        included={included}
        onAck={onAck}
        onInclude={onInclude}
      />
    </Box>
  )
}
