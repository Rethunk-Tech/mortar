import { useLingui } from '@lingui/react/macro'
import { Box, Chip, Typography } from '@mui/material'
import { type ReactNode, useEffect, useState } from 'react'
import { Details as ReadDetails } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'
import type { Changelog } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexus/models.ts'
import type { Details } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/source/models.ts'
import { parseBBCode } from '../mods/bbcode.ts'
import { ChangelogEntries } from '../mods/ChangelogDialog.tsx'
import { DependencyChips } from '../mods/DependencyChips.tsx'
import { DetailsAside } from '../mods/DetailsAside.tsx'
import { DetailsHeader } from '../mods/DetailsHeader.tsx'
import { type Dependency, nexusDependencies } from '../mods/dependencies.ts'
import { MarkdownView } from '../mods/MarkdownView.tsx'
import { Rich } from '../mods/NexusDetails.tsx'
import { loadDetails, useNexusEntry } from '../mods/nexusDetails.ts'
import { heading } from '../mods/paper.ts'
import { useReleases } from '../mods/releases.ts'
import { ErrorRetry } from '../shell/ErrorRetry.tsx'
import { SkeletonRows } from '../shell/SkeletonRows.tsx'
import { useLoaded } from '../shell/useLoaded.ts'
import { type InlineError, inlineError } from '../toasts/report.ts'
import { GITHUB, NEXUS } from './browseConstants.ts'
import type { BrowseItem } from './browseTypes.ts'
import { CardPicture } from './ResultCard.tsx'
import { SourceBadges } from './SourceBadges.tsx'
import { hitKey, useBrowseSelection } from './selection.ts'
import { useStats } from './stats.ts'
import { type ActionProps, useCardAction } from './useCardAction.tsx'

const VERSIONS_SHOWN = 6
const PICTURE = 52
const SKELETON_ROWS = 4
const SKELETON_PX = 18
const text = { fontSize: 13 } as const

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Typography sx={heading}>{title}</Typography>
      {children}
    </Box>
  )
}

function Versions({ versions }: { versions: string[] }) {
  const { t } = useLingui()
  if (versions.length === 0) {
    return null
  }
  const [latest, ...older] = versions
  const shown = older.slice(0, VERSIONS_SHOWN - 1)
  const count = older.length - shown.length
  return (
    <Section title={t`Versions`}>
      <Typography sx={text}>{t`${latest} (latest)`}</Typography>
      {shown.length > 0 ? (
        <Typography sx={{ ...text, color: 'text.secondary' }}>
          {shown.join(', ')}
          {count > 0 ? `, ${t`and ${count} more`}` : ''}
        </Typography>
      ) : null}
    </Section>
  )
}

function Chips({ title, names }: { title: string; names: string[] }) {
  return names.length > 0 ? (
    <Section title={title}>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
        {names.map((n) => (
          <Chip key={n} size="small" variant="outlined" label={n} />
        ))}
      </Box>
    </Section>
  ) : null
}

/** What every source's page comes down to; empty parts are left out. */
interface Page {
  categories: string[]
  description: ReactNode
  versions: string[]
  dependencies: Dependency[]
  changelog: ReactNode
}

function PageSections({ page }: { page: Page }) {
  const { t } = useLingui()
  return (
    <>
      <Chips title={t`Categories`} names={page.categories} />
      {page.description ? <Section title={t`Description`}>{page.description}</Section> : null}
      <Versions versions={page.versions} />
      {page.dependencies.length > 0 ? (
        <Section title={t`Dependencies`}>
          <DependencyChips deps={page.dependencies} />
        </Section>
      ) : null}
      {page.changelog ? <Section title={t`Changelog`}>{page.changelog}</Section> : null}
    </>
  )
}

function Loading() {
  const { t } = useLingui()
  return (
    <SkeletonRows label={t`Reading the mod's page…`} count={SKELETON_ROWS} height={SKELETON_PX} />
  )
}

const changelogOf = (logs: Changelog[] | null | undefined) =>
  logs && logs.length > 0 ? <ChangelogEntries logs={logs} installed="" /> : null

// A Nexus mod's page comes from the Nexus details the Mods tab reads too: one read per mod per session.
function NexusPage({ modId }: { modId: number }) {
  const entry = useNexusEntry(modId)
  const [attempt, setAttempt] = useState(0)
  useEffect(() => {
    if (attempt >= 0) {
      loadDetails(modId).catch(() => undefined)
    }
  }, [modId, attempt])
  const details = entry?.details
  if (!details && entry?.error) {
    return (
      <ErrorRetry
        error={{ message: entry.error, details: entry.error }}
        onRetry={() => setAttempt((n) => n + 1)}
      />
    )
  }
  if (!details) {
    return <Loading />
  }
  const files = (details.files ?? []).filter((f) => f.category !== 'ARCHIVED')
  const versions = [
    ...new Set([details.page.version, ...files.map((f) => f.version)].filter(Boolean)),
  ]
  const { description } = details.page
  return (
    <PageSections
      page={{
        categories: details.category ? [details.category] : [],
        description: description ? <Rich blocks={parseBBCode(description)} /> : null,
        versions,
        dependencies: nexusDependencies(details.page.requirements),
        changelog: changelogOf(details.changelogs),
      }}
    />
  )
}

type Read = { details: Details } | { error: InlineError } | null

// One read per open; the browse service keeps each answer for the session.
function useSiteDetails(game: string, source: string, id: string) {
  const { data, error, reload } = useLoaded<Read>(
    () => ReadDetails(game, source, id).then((details) => ({ details })),
    [game, source, id],
    null,
  )
  const state: Read = error === null ? data : { error: inlineError(error) }
  return { state, retry: reload }
}

// Thunderstore and GitHub pages: the README and what the site lists; a GitHub repo's releases are its versions and
// changelog.
function SitePage({ game, item }: { game: string; item: BrowseItem }) {
  const { state, retry } = useSiteDetails(game, item.source, item.id)
  const releases = useReleases(item.source === GITHUB, item.source === GITHUB ? item.id : '')
  if (state !== null && 'error' in state) {
    return <ErrorRetry error={state.error} onRetry={retry} />
  }
  if (state === null) {
    return <Loading />
  }
  const { details } = state
  const logs = releases.state !== null && 'logs' in releases.state ? releases.state.logs : []
  const changelog = details.changelog ? <MarkdownView source={details.changelog} /> : null
  return (
    <PageSections
      page={{
        categories: details.categories ?? [],
        description: details.description ? (
          <MarkdownView source={details.description} />
        ) : (
          <Typography sx={text}>{item.summary}</Typography>
        ),
        versions: item.source === GITHUB ? logs.map((l) => l.version) : (details.versions ?? []),
        dependencies: (details.dependencies ?? []).map((name) => ({ name })),
        changelog: item.source === GITHUB ? changelogOf(logs) : changelog,
      }}
    />
  )
}

function Panel({
  item,
  initial,
  card,
  game,
  sourceNames,
}: {
  item: BrowseItem
  initial: string
  card: ActionProps
  game: string
  sourceNames: Map<string, string>
}) {
  const sources = [item.source, ...(item.alts ?? []).map((a) => a.source)]
  const [picked, setPicked] = useState(initial)
  const source = sources.includes(picked) ? picked : item.source
  const { shownItem, action } = useCardAction(card, item, source)
  const stats = useStats(shownItem)
  return (
    <Box sx={{ p: 1.75, display: 'flex', flexDirection: 'column', gap: 1.25, minHeight: '100%' }}>
      <DetailsHeader
        picture={<CardPicture picture={item.picture} size={PICTURE} dim={1} />}
        title={item.name}
        subtitle={item.author}
        onClose={() => useBrowseSelection.getState().select(null)}
      />
      <SourceBadges sources={sources} picked={source} names={sourceNames} onPick={setPicked} />
      <Box sx={{ alignSelf: 'flex-start' }}>{action}</Box>
      <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>{stats}</Typography>
      {source === NEXUS ? (
        <NexusPage key={shownItem.id} modId={Number(shownItem.id)} />
      ) : (
        <SitePage key={`${source}:${shownItem.id}`} game={game} item={shownItem} />
      )}
    </Box>
  )
}

/** The details panel of the Browse hit the player clicked, beside the results. */
export function BrowseDetails({
  game,
  card,
  sourceNames,
}: {
  game: string
  card: ActionProps
  sourceNames: Map<string, string>
}) {
  const { t } = useLingui()
  const selected = useBrowseSelection((s) => s.selected)
  return (
    <DetailsAside
      open={selected !== null}
      label={t`Selected mod`}
      onClose={() => useBrowseSelection.getState().select(null)}
    >
      {selected ? (
        <Panel
          key={hitKey(selected.item)}
          item={selected.item}
          initial={selected.source}
          card={card}
          game={game}
          sourceNames={sourceNames}
        />
      ) : null}
    </DetailsAside>
  )
}
