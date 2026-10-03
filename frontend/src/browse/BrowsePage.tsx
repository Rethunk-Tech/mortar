import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react'
import {
  Box,
  Button,
  Card,
  CardContent,
  CardMedia,
  Chip,
  Pagination,
  Stack,
  Tab,
  Tabs,
  TextField,
  Typography,
} from '@mui/material'
import { useEffect, useState } from 'react'

import { clampPage, DEBOUNCE_MS, PAGE_SIZE } from './browseState.ts'
import type { BrowseItem, BrowsePageProps } from './browseTypes.ts'

const NEXUS = 'nexus'
const GITHUB = 'github'
const FIRST_PAGE = 1
const CARD_IMAGE_HEIGHT = 140

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
  const { i18n } = useLingui()
  const [source, setSource] = useState(NEXUS)
  const [draft, setDraft] = useState('')
  const [text, setText] = useState('')
  const [page, setPage] = useState(FIRST_PAGE)
  const [result, setResult] = useState({ total: 0, items: [] as BrowseItem[] })
  const [busy, setBusy] = useState('')

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
    if (text.trim() === '') {
      setResult({ total: 0, items: [] })
      setBusy('')
      return
    }
    let cancelled = false
    search({ game, source, text, page, profileID })
      .then((next) => {
        if (!cancelled) {
          setResult(next)
          setBusy('')
          setPage((current) => clampPage({ page: current, total: next.total }))
        }
      })
      .catch((err: { message?: string }) => {
        if (!cancelled) {
          const { message } = err
          setBusy(message ?? i18n._(msg`GitHub is busy, try again in a minute`))
        }
      })
    return () => {
      cancelled = true
    }
  }, [game, source, text, page, profileID, search, i18n])

  const pageCount = Math.max(FIRST_PAGE, Math.ceil(result.total / PAGE_SIZE) || FIRST_PAGE)

  return (
    <Stack spacing={2} sx={{ p: 2, overflow: 'auto' }}>
      <Typography variant="h5">{i18n._(msg`Browse`)}</Typography>
      <TextField
        autoFocus={true}
        fullWidth={true}
        label={i18n._(msg`Search mods`)}
        value={draft}
        onChange={(event) => {
          const { value } = event.target
          setDraft(value)
        }}
      />
      <Tabs
        value={source}
        onChange={(_event, next: string) => {
          setSource(next)
          setPage(FIRST_PAGE)
        }}
      >
        <Tab value={NEXUS} label={i18n._(msg`Nexus`)} />
        <Tab value={GITHUB} label={i18n._(msg`GitHub`)} />
        {hasCurseForgeKey ? <Tab value="curseforge" label={i18n._(msg`CurseForge`)} /> : null}
      </Tabs>
      {busy === '' ? null : <Typography color="text.secondary">{busy}</Typography>}
      <Stack spacing={2}>
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
      </Stack>
      {result.total > PAGE_SIZE ? (
        <Pagination
          count={pageCount}
          page={page}
          onChange={(_event, next) => {
            setPage(next)
          }}
        />
      ) : null}
    </Stack>
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
  const { i18n } = useLingui()
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
  return (
    <Card>
      <Stack direction="row">
        {picture === '' ? (
          <Box sx={{ width: CARD_IMAGE_HEIGHT, height: CARD_IMAGE_HEIGHT }} />
        ) : (
          <CardMedia
            component="img"
            image={picture}
            alt=""
            sx={{ width: CARD_IMAGE_HEIGHT, height: CARD_IMAGE_HEIGHT }}
          />
        )}
        <CardContent sx={{ flex: 1 }}>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
            <Typography variant="h6">{name}</Typography>
            {installed ? <Chip size="small" label={i18n._(msg`In this profile`)} /> : null}
          </Stack>
          <Typography variant="body2" color="text.secondary">
            {author}
          </Typography>
          <Typography variant="body2">{summary}</Typography>
          <Typography variant="caption">
            {source === NEXUS
              ? i18n._(msg`${endorsements} endorsements · ${downloads} downloads`)
              : i18n._(msg`${stars} stars`)}
          </Typography>
          <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
            {source === NEXUS ? (
              <>
                <Button size="small" onClick={() => openUrl(url)}>
                  {i18n._(msg`Open on Nexus`)}
                </Button>
                {premium ? (
                  <Button size="small" onClick={() => downloadNexus(id)}>
                    {i18n._(msg`Download`)}
                  </Button>
                ) : null}
              </>
            ) : (
              <Button size="small" onClick={() => addGitHub(id)}>
                {i18n._(msg`Add`)}
              </Button>
            )}
          </Stack>
        </CardContent>
      </Stack>
    </Card>
  )
}

export { BrowsePage }
