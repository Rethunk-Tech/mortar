import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Card,
  Chip,
  InputAdornment,
  Pagination,
  Skeleton,
  TextField,
  Typography,
} from '@mui/material'
import { CloudOff, Download, ExternalLink, Plus, Search, SearchX } from 'lucide-react'
import { useEffect, useState } from 'react'
import { i18n } from '../i18n/index.ts'
import { PrefSegmented } from '../settings/PrefControls.tsx'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { IconAction } from '../shell/IconAction.tsx'
import { usePending } from '../toasts/usePending.ts'
import { clampPage, DEBOUNCE_MS, PAGE_SIZE } from './browseState.ts'
import type { BrowseItem, BrowsePageProps } from './browseTypes.ts'

const NEXUS = 'nexus'
const GITHUB = 'github'
const CURSEFORGE = 'curseforge'
const FIRST_PAGE = 1
const PICTURE_PX = 72
const CARD_MIN_PX = 340
const SKELETON_KEYS = ['a', 'b', 'c', 'd', 'e', 'f']
const ICON_SIZE = 40
const STALE_OPACITY = 0.6

function searchHint(source: string, premium: boolean): string {
  if (source === GITHUB) {
    return i18n._(
      msg`Search GitHub for mods published as releases. Add puts the latest release in this profile.`,
    )
  }
  if (source === CURSEFORGE) {
    return i18n._(msg`Search CurseForge. Open the mod's page to download it.`)
  }
  if (source === NEXUS && premium) {
    return i18n._(msg`Search Nexus Mods. Download installs the mod into this profile.`)
  }
  return i18n._(
    msg`Search Nexus Mods. Free accounts download from the mod's page with Mod Manager Download.`,
  )
}

function openPageLabel(source: string): string {
  if (source === GITHUB) {
    return i18n._(msg`Open on GitHub`)
  }
  if (source === CURSEFORGE) {
    return i18n._(msg`Open on CurseForge`)
  }
  return i18n._(msg`Open on Nexus`)
}

const grid = {
  display: 'grid',
  gridTemplateColumns: `repeat(auto-fill, minmax(${CARD_MIN_PX}px, 1fr))`,
  gap: '6px',
} as const

type Status = 'idle' | 'loading' | 'done' | 'error'

function useBrowseQuery({
  game,
  profileID,
  search,
}: Pick<BrowsePageProps, 'game' | 'profileID' | 'search'>) {
  const [source, setSource] = useState(NEXUS)
  const [draft, setDraft] = useState('')
  const [text, setText] = useState('')
  const [page, setPage] = useState(FIRST_PAGE)
  const [retry, setRetry] = useState(0)
  const [result, setResult] = useState({ total: 0, items: [] as BrowseItem[] })
  const [status, setStatus] = useState<Status>('idle')
  const [error, setError] = useState('')

  useEffect(() => {
    const timer = setTimeout(() => {
      setText(draft)
      setPage(FIRST_PAGE)
    }, DEBOUNCE_MS)
    return () => {
      clearTimeout(timer)
    }
  }, [draft])

  useEffect(() => {
    if (text.trim() === '' || retry < 0) {
      setResult({ total: 0, items: [] })
      setStatus('idle')
      return
    }
    let cancelled = false
    setStatus('loading')
    search({ game, source, text, page, profileID })
      .then((next) => {
        if (!cancelled) {
          setResult(next)
          setStatus('done')
          setPage((current) => clampPage({ page: current, total: next.total }))
        }
      })
      .catch((err: { message?: string }) => {
        if (!cancelled) {
          setError(err.message ?? '')
          setStatus('error')
        }
      })
    return () => {
      cancelled = true
    }
  }, [game, source, text, page, profileID, search, retry])

  return {
    source,
    setSource,
    draft,
    setDraft,
    text,
    page,
    setPage,
    setRetry,
    result,
    status,
    error,
  }
}

function BrowsePage({
  game,
  profileID,
  premium,
  hasCurseForgeKey,
  search,
  openUrl,
  downloadNexus,
  addGitHub,
}: BrowsePageProps) {
  const { t } = useLingui()
  const {
    source,
    setSource,
    draft,
    setDraft,
    text,
    page,
    setPage,
    setRetry,
    result,
    status,
    error,
  } = useBrowseQuery({ game, profileID, search })
  const pageCount = Math.max(FIRST_PAGE, Math.ceil(result.total / PAGE_SIZE) || FIRST_PAGE)
  const sources = [
    { value: NEXUS, label: t`Nexus Mods` },
    { value: GITHUB, label: t`GitHub` },
    ...(hasCurseForgeKey ? [{ value: CURSEFORGE, label: t`CurseForge` }] : []),
  ]
  const placeholder = source === GITHUB ? t`Search GitHub releases` : t`Search Nexus Mods`
  const hint = searchHint(source, premium)
  let body: React.ReactNode
  if (status === 'idle') {
    body = (
      <EmptyState icon={<Search size={ICON_SIZE} />} title={t`Find mods to add`}>
        {hint}
      </EmptyState>
    )
  } else if (status === 'error') {
    body = (
      <EmptyState
        icon={<CloudOff size={ICON_SIZE} />}
        title={t`Search did not work`}
        action={
          <Button variant="outlined" onClick={() => setRetry((n) => n + 1)}>
            {t`Try again`}
          </Button>
        }
      >
        {error || t`The service may be busy. Try again in a minute.`}
      </EmptyState>
    )
  } else if (status === 'loading' && result.items.length === 0) {
    body = (
      <Box sx={grid}>
        {SKELETON_KEYS.map((key) => (
          <Skeleton key={key} variant="rounded" height={PICTURE_PX + 24} />
        ))}
      </Box>
    )
  } else if (result.items.length === 0) {
    body = (
      <EmptyState icon={<SearchX size={ICON_SIZE} />} title={t`No mods match "${text}"`}>
        {t`Try fewer words, or the mod's exact name.`}
      </EmptyState>
    )
  } else {
    body = (
      <>
        <Typography sx={{ fontSize: 13, color: 'text.secondary', mb: 1 }}>
          {plural(result.total, { one: '# result', other: '# results' })}
        </Typography>
        <Box sx={{ ...grid, opacity: status === 'loading' ? STALE_OPACITY : 1 }}>
          {result.items.map((item) => (
            <ResultCard
              key={`${item.source}:${item.id}`}
              item={item}
              premium={premium}
              openUrl={openUrl}
              downloadNexus={downloadNexus}
              addGitHub={addGitHub}
            />
          ))}
        </Box>
        {result.total > PAGE_SIZE ? (
          <Pagination
            count={pageCount}
            page={page}
            onChange={(_event, next) => setPage(next)}
            sx={{ display: 'flex', justifyContent: 'center', mt: 2 }}
          />
        ) : null}
      </>
    )
  }

  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, px: 2, pt: 1.25, pb: 0.75 }}>
        <PrefSegmented
          value={source}
          label={t`Source`}
          options={sources}
          onChange={(next) => {
            setSource(next)
            setPage(FIRST_PAGE)
          }}
        />
        <TextField
          size="small"
          autoFocus={true}
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          placeholder={placeholder}
          slotProps={{
            htmlInput: { 'aria-label': placeholder },
            input: {
              startAdornment: (
                <InputAdornment position="start">
                  <Search size={14} />
                </InputAdornment>
              ),
              sx: {
                height: 36,
                fontSize: 13,
                borderRadius: '6px',
                bgcolor: 'var(--mortar-overlay-30)',
                '& .MuiOutlinedInput-notchedOutline': { borderColor: 'var(--mortar-hairline-15)' },
              },
            },
          }}
          sx={{ flex: 1, minWidth: 0 }}
        />
      </Box>
      <Box
        sx={{
          flex: 1,
          minHeight: 0,
          overflowY: 'auto',
          px: 2,
          py: 1,
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        {body}
      </Box>
    </Box>
  )
}

function ResultCard({
  item,
  premium,
  openUrl,
  downloadNexus,
  addGitHub,
}: {
  item: BrowseItem
  premium: boolean
  openUrl: (url: string) => void
  downloadNexus: (modID: string) => void
  addGitHub: (repo: string) => void
}) {
  const { t } = useLingui()
  const [pending, run] = usePending()
  const {
    source,
    id,
    name,
    summary,
    author,
    picture,
    endorsements,
    stars,
    downloads,
    url,
    installed,
  } = item
  const stats =
    source === NEXUS
      ? t`${endorsements} endorsements · ${downloads} downloads`
      : plural(stars, { one: '# star', other: '# stars' })
  let action: React.ReactNode = null
  if (installed) {
    action = <Chip size="small" label={t`In this profile`} />
  } else if (source === GITHUB) {
    action = (
      <Button
        size="small"
        variant="contained"
        startIcon={<Plus size={14} />}
        disabled={pending}
        onClick={() => run(() => Promise.resolve(addGitHub(id)))}
      >
        {t`Add`}
      </Button>
    )
  } else if (source === NEXUS && premium) {
    action = (
      <Button
        size="small"
        variant="contained"
        startIcon={<Download size={14} />}
        disabled={pending}
        onClick={() => run(() => Promise.resolve(downloadNexus(id)))}
      >
        {t`Download`}
      </Button>
    )
  } else if (source === NEXUS) {
    action = (
      <DisabledReason title={t`Premium only`} disabled={true}>
        <Button size="small" variant="contained" startIcon={<Download size={14} />} disabled={true}>
          {t`Download`}
        </Button>
      </DisabledReason>
    )
  }
  return (
    <Card sx={{ display: 'flex', gap: 1.25, p: 1, borderRadius: '6px', minWidth: 0 }}>
      {picture === '' ? (
        <Box
          sx={{
            width: PICTURE_PX,
            height: PICTURE_PX,
            flexShrink: 0,
            borderRadius: '4px',
            bgcolor: 'var(--mortar-raised)',
          }}
        />
      ) : (
        <Box
          component="img"
          src={picture}
          alt=""
          loading="lazy"
          sx={{
            width: PICTURE_PX,
            height: PICTURE_PX,
            flexShrink: 0,
            objectFit: 'cover',
            borderRadius: '4px',
          }}
        />
      )}
      <Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: 0.25 }}>
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
            WebkitLineClamp: 2,
            WebkitBoxOrient: 'vertical',
            overflow: 'hidden',
          }}
        >
          {summary}
        </Typography>
      </Box>
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'flex-end',
          justifyContent: 'space-between',
          gap: 0.5,
        }}
      >
        <IconAction
          label={openPageLabel(source)}
          icon={<ExternalLink size={15} />}
          onClick={() => openUrl(url)}
        />
        {action}
      </Box>
    </Card>
  )
}

export { BrowsePage }
