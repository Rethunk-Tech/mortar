import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { Changelog as SourceChangelog } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'
import type { Changelog } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nexus/models.ts'
import { UpdateChangelog } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nexussvc/service.ts'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { useProfiles } from '../../profiles/store.ts'
import { PageLink } from '../../share/CollectionNotes.tsx'
import { parseInstructions } from '../../share/instructions.ts'
import { Fold } from '../../shell/Fold.tsx'
import { useLoaded } from '../../shell/useLoaded.ts'
import { changelogNoteIsRisky } from '../changelogRange.ts'
import { MarkdownView } from '../MarkdownView.tsx'

function ChangelogText({ text, risky }: { text: string; risky?: boolean }) {
  return (
    <Box
      sx={{
        fontSize: 'base.fontSize',
        overflowWrap: 'anywhere',
        color: risky ? 'warning.main' : undefined,
      }}
    >
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

// A version's notes: its Markdown body, then its listed notes with risky ones marked.
function ChangelogEntryBody({ entry }: { entry: Changelog }) {
  return (
    <>
      {entry.body ? <MarkdownView source={entry.body} /> : null}
      {(entry.notes ?? []).map((n) => (
        <ChangelogText key={n} text={`• ${n}`} risky={changelogNoteIsRisky(n)} />
      ))}
    </>
  )
}

function Version({ entry }: { entry: Changelog }) {
  return (
    <Box>
      <Typography variant="subtitle2">
        {entry.date ? `${entry.version} · ${entry.date}` : entry.version}
      </Typography>
      <ChangelogEntryBody entry={entry} />
    </Box>
  )
}

// Mounted only while the disclosure is open, so nothing is fetched for a row nobody expands.
function Loaded({ update }: { update: Update }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const { data, error } = useLoaded<Changelog[] | undefined>(
    () =>
      (update.nexusId > 0 || update.githubRepo || !update.package
        ? UpdateChangelog(game, update.nexusId, update.githubRepo, update.installed, update.version)
        : SourceChangelog(
            game,
            update.packageSource || 'thunderstore',
            update.package,
            update.installed,
            update.version,
          )
      ).then((got) => got ?? []),
    [
      game,
      update.githubRepo,
      update.installed,
      update.nexusId,
      update.package,
      update.packageSource,
      update.version,
    ],
    undefined,
  )
  const logs = error === null ? data : 'failed'
  if (logs === undefined) {
    return <Box role="status" aria-busy={true} sx={{ minHeight: 20 }} />
  }
  if (logs === 'failed') {
    return <Typography sx={{ fontSize: 'small.fontSize' }}>{t`Changelog unavailable`}</Typography>
  }
  if (logs.length === 0) {
    return (
      <Typography sx={{ fontSize: 'small.fontSize' }}>{t`No changelog for this update`}</Typography>
    )
  }
  return logs.map((c) => <Version key={c.version} entry={c} />)
}

export function Changes({ update }: { update: Update }) {
  const { t } = useLingui()
  if (!(update.nexusId > 0 || update.githubRepo || update.package)) {
    return null
  }
  return (
    <Fold title={t`What is new`}>
      <Loaded update={update} />
    </Fold>
  )
}

export { ChangelogEntryBody }
