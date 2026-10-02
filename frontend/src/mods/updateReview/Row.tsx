import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { Browser } from '@wailsio/runtime'
import { ArrowRight, ExternalLink } from 'lucide-react'
import type { Update } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { useProfiles } from '../../profiles/store.ts'
import { useQueue } from '../../queue/store.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { changelogsBetween, changelogsHaveRiskyNotes } from '../changelogRange.ts'
import { sameId, siblingsOf } from '../lookup.ts'
import { useNexusDetails } from '../nexusDetails.ts'
import { LetterTile } from '../parts.tsx'
import { useMods } from '../store.ts'
import { ROW_TILE } from './constants.ts'
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
  onUpdateAll,
  picture,
}: {
  update: Update
  profileId: string
  caution: string
  acked: boolean
  included: boolean
  onAck: (on: boolean) => void
  onInclude: (on: boolean) => void
  onUpdateAll: () => void
  picture?: string
}) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const mod = mods.find((m) => m.key === update.key && sameId(m.uniqueId, update.uniqueId))
  const entry = useProfiles((s) =>
    s.profiles.find((p) => p.id === profileId)?.entries?.find((e) => e.key === update.key),
  )
  const details = useNexusDetails((s) => s.byId[update.nexusId]?.details)
  const riskyChangelog =
    update.nexusId > 0 && details
      ? changelogsHaveRiskyNotes(
          changelogsBetween(details.changelogs ?? [], update.installed, update.version),
        )
      : false
  const queued = useQueue((s) => pendingUpdate(s.state.items, profileId, update))
  const notes = [
    ...(mod ? siblingsOf(mods, mod).map((o) => t`Also updates ${o.name} (same download)`) : []),
    ...(mod && !mod.enabled ? [t`Switched off in this profile`] : []),
    ...(update.unofficial ? [t`Unofficial`] : []),
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
        borderBottom: '1px solid rgba(255,255,255,0.08)',
      }}
    >
      <LetterTile
        mod={{ uniqueId: update.uniqueId, name: update.name, ...(picture ? { picture } : {}) }}
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
        </Box>
      </Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        {update.url ? (
          <Button
            variant="outlined"
            endIcon={<ExternalLink size={12} />}
            onClick={() => Browser.OpenURL(update.url).catch(reportUnexpected)}
            sx={{ whiteSpace: 'nowrap' }}
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
          onUpdateAll={onUpdateAll}
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
