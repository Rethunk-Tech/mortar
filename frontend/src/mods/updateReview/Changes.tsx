import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import type { Changelog } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nexus/models.ts'
import { UpdateChangelog } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { useProfiles } from '../../profiles/store.ts'
import { PageLink } from '../../share/CollectionNotes.tsx'
import { parseInstructions } from '../../share/instructions.ts'
import { Fold } from '../../shell/Fold.tsx'
import { changelogNoteIsRisky } from '../changelogRange.ts'

function ChangelogText({ text, risky }: { text: string; risky?: boolean }) {
  return (
    <Box sx={{ fontSize: 13, overflowWrap: 'anywhere', color: risky ? 'warning.main' : undefined }}>
      {parseInstructions(text).map(({ at, parts }) => (
        <Box key={at} sx={{ minHeight: '1.4em' }}>
          {parts.map((part) => (
            <PageLink key={part.at} part={part} />
          ))}
        </Box>
      ))}
    </Box>
  )
}

function Version({ entry }: { entry: Changelog }) {
  return (
    <Box>
      <Typography variant="subtitle2">
        {entry.date ? `${entry.version} · ${entry.date}` : entry.version}
      </Typography>
      {entry.body ? <ChangelogText text={entry.body} /> : null}
      {(entry.notes ?? []).map((n) => (
        <ChangelogText key={n} text={`• ${n}`} risky={changelogNoteIsRisky(n)} />
      ))}
    </Box>
  )
}

// Mounted only while the disclosure is open, so nothing is fetched for a row nobody expands.
function Loaded({ update }: { update: Update }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const [logs, setLogs] = useState<Changelog[] | 'failed' | undefined>(undefined)
  useEffect(() => {
    let cancelled = false
    UpdateChangelog(game, update.nexusId, update.githubRepo, update.installed, update.version).then(
      (got) => !cancelled && setLogs(got ?? []),
      () => !cancelled && setLogs('failed'),
    )
    return () => {
      cancelled = true
    }
  }, [game, update.githubRepo, update.installed, update.nexusId, update.version])
  if (logs === undefined) {
    return <Box role="status" aria-busy={true} sx={{ minHeight: 20 }} />
  }
  if (logs === 'failed') {
    return <Typography sx={{ fontSize: 12 }}>{t`Changelog unavailable`}</Typography>
  }
  if (logs.length === 0) {
    return <Typography sx={{ fontSize: 12 }}>{t`No changelog for this update`}</Typography>
  }
  return logs.map((c) => <Version key={c.version} entry={c} />)
}

export function Changes({ update }: { update: Update }) {
  const { t } = useLingui()
  if (!(update.nexusId > 0 || update.githubRepo)) {
    return null
  }
  return (
    <Fold title={t`What's new`}>
      <Loaded update={update} />
    </Fold>
  )
}

export { ChangelogText }
