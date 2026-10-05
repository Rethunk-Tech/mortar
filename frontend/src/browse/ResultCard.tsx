import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Card, Chip, Typography } from '@mui/material'
import { ExternalLink } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useQueue } from '../queue/store.ts'
import { IconAction } from '../shell/IconAction.tsx'
import { usePending } from '../toasts/usePending.ts'
import {
  GITHUB,
  GRAY_OPACITY,
  NEXUS,
  openPageLabel,
  PICTURE_PX,
  ROW_PICTURE_PX,
  THUNDERSTORE,
} from './browseConstants.ts'
import type { BrowseModes } from './browseModes.ts'
import type { BrowseItem, ResultCardProps } from './browseTypes.ts'
import { CardAction } from './CardAction.tsx'
import { cardState, isActive, shownState } from './cardState.ts'

// One badge per source the mod is on; the filled one is where Add installs from.
function SourceBadges({
  sources,
  picked,
  names,
  onPick,
}: {
  sources: string[]
  picked: string
  names: Map<string, string>
  onPick: (source: string) => void
}) {
  if (sources.length < 2) {
    return null
  }
  return (
    <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap', pt: 0.25 }}>
      {sources.map((id) => (
        <Chip
          key={id}
          size="small"
          label={names.get(id) ?? id}
          variant={id === picked ? 'filled' : 'outlined'}
          onClick={() => onPick(id)}
        />
      ))}
    </Box>
  )
}

function CardPicture({ picture, size, dim }: { picture: string; size: number; dim: number }) {
  const sx = { width: size, height: size, flexShrink: 0, borderRadius: '4px', opacity: dim }
  if (picture === '') {
    return <Box sx={{ ...sx, bgcolor: 'var(--mortar-raised)' }} />
  }
  return (
    <Box component="img" src={picture} alt="" loading="lazy" sx={{ ...sx, objectFit: 'cover' }} />
  )
}

function useStats(item: BrowseItem): string {
  const { t } = useLingui()
  const { endorsements, stars, downloads } = item
  return item.source === NEXUS
    ? t`${plural(endorsements, { one: '# endorsement', other: '# endorsements' })} · ${plural(downloads, { one: '# download', other: '# downloads' })}`
    : plural(stars, { one: '# star', other: '# stars' })
}

// Gray out dims the mod's picture and text; the action and its chip stay readable.
function isGray(modes: BrowseModes, item: BrowseItem, installed: boolean): boolean {
  return (
    (modes.installed === 'gray' && installed) ||
    (modes.obsolete === 'gray' && item.obsolete) ||
    (modes.broken === 'gray' && item.broken)
  )
}

function ResultCard(props: ResultCardProps) {
  const { row, item, premium, openUrl, profileID, modes, sourceNames } = props
  const [pending, run] = usePending()
  const [openedFiles, setOpenedFiles] = useState(false)
  const items = useQueue((s) => s.state.items)
  const { name, summary, author } = item
  // The same mod found on several sources is one card; its first source is the default and a badge picks another.
  const primary = { source: item.source, id: item.id, url: item.url, installed: item.installed }
  const choices = [primary, ...(item.alts ?? [])]
  const [picked, setPicked] = useState(item.source)
  const { source, id, url } = choices.find((c) => c.source === picked) ?? primary
  const installed = choices.some((c) => c.installed)
  const stats = useStats(item)
  const live = cardState(items, source, id, profileID)
  // A free account's click on Mod Manager Download is not ours to see; the card waits for its nxm item.
  const queueState =
    live.kind === 'idle' && openedFiles ? ({ kind: 'waiting-nexus' } as const) : live
  const [watched, setWatched] = useState(false)
  useEffect(() => {
    if (isActive(queueState)) {
      setWatched(true)
    }
  }, [queueState])
  const shown = shownState(queueState, installed, watched)
  const add = () => {
    if (source === GITHUB) {
      return props.addGitHub(id)
    }
    return source === THUNDERSTORE ? props.addPackage(id) : props.addDirect(source, id)
  }
  const dim = isGray(modes, item, installed) ? GRAY_OPACITY : 1
  return (
    <Card
      sx={{
        display: 'flex',
        alignItems: row ? 'center' : 'stretch',
        gap: 1.25,
        p: 1,
        borderRadius: '6px',
        minWidth: 0,
      }}
    >
      <CardPicture picture={item.picture} size={row ? ROW_PICTURE_PX : PICTURE_PX} dim={dim} />
      <Box
        sx={{
          flex: 1,
          minWidth: 0,
          display: 'flex',
          flexDirection: 'column',
          gap: 0.25,
          opacity: dim,
        }}
      >
        <Typography noWrap={true} title={name} sx={{ fontSize: 15, fontWeight: 600 }}>
          {name}
        </Typography>
        <Typography noWrap={true} sx={{ fontSize: 12, color: 'text.secondary' }}>
          {author === '' ? stats : `${author} · ${stats}`}
        </Typography>
        <Typography
          title={summary}
          sx={{
            fontSize: 13,
            color: 'var(--mortar-ink-soft)',
            display: '-webkit-box',
            WebkitLineClamp: row ? 1 : 2,
            WebkitBoxOrient: 'vertical',
            overflow: 'hidden',
          }}
        >
          {summary}
        </Typography>
        <SourceBadges
          sources={choices.map((c) => c.source)}
          picked={source}
          names={sourceNames}
          onPick={setPicked}
        />
      </Box>
      <Box
        sx={{
          display: 'flex',
          flexDirection: row ? 'row' : 'column',
          alignItems: row ? 'center' : 'flex-end',
          justifyContent: 'space-between',
          gap: 0.5,
        }}
      >
        <IconAction
          label={openPageLabel(source)}
          icon={<ExternalLink size={15} />}
          onClick={() => openUrl(url)}
        />
        <CardAction
          item={item}
          source={source}
          state={shown}
          installed={installed}
          premium={premium}
          pending={pending}
          onAdd={() => run(() => Promise.resolve(add()))}
          onDownload={() => run(() => Promise.resolve(props.downloadNexus(id)))}
          onOpenFiles={() => {
            setOpenedFiles(true)
            openUrl(`${url}?tab=files`)
          }}
        />
      </Box>
    </Card>
  )
}

export { ResultCard }
