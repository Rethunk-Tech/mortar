import { useLingui } from '@lingui/react/macro'
import { Box, Button, ButtonBase, Dialog, Typography } from '@mui/material'
import { Camera, FolderOpen } from 'lucide-react'
import { type ReactNode, useCallback, useEffect, useState } from 'react'
import {
  OpenScreenshotsFolder,
  ScreenshotFile,
  Screenshots,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import { SetCover } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import type {
  Gallery,
  Shot,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/shots/models.ts'
import { PageActions } from '../game/PageActions.tsx'
import { storeName } from '../games/storeName.ts'
import { When } from '../i18n/When.tsx'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { space } from '../theme/density.ts'
import { errorMessage, reportError, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

const TILE_MIN_PX = 220
const STEAM = 'steam'

const shotUrl = (game: string, s: Shot) =>
  `/screenshot/${encodeURIComponent(game)}/${s.source}/${encodeURIComponent(s.name)}`

function Viewer({
  game,
  shots,
  index,
  onIndex,
  onClose,
}: {
  game: string
  shots: Shot[]
  index: number
  onIndex: (i: number) => void
  onClose: () => void
}) {
  const { t } = useLingui()
  const profile = useProfiles(openProfileOf)
  const shot = shots[index]
  if (!shot) {
    return null
  }
  const makeCover = async () => {
    if (!profile) {
      return
    }
    try {
      const file = await ScreenshotFile(game, shot.source, shot.name)
      useProfiles.getState().replace(await SetCover(game, profile.id, file))
      useToasts.getState().push({ kind: 'success', title: t`${profile.name} has a new cover` })
    } catch (e) {
      reportError(t`Could not set the cover`)(e)
    }
  }
  return (
    <Dialog
      open={true}
      onClose={onClose}
      maxWidth="xl"
      onKeyDown={(e) => {
        if (e.key === 'ArrowRight' && index < shots.length - 1) {
          onIndex(index + 1)
        } else if (e.key === 'ArrowLeft' && index > 0) {
          onIndex(index - 1)
        }
      }}
    >
      <Box
        component="img"
        src={shotUrl(game, shot)}
        alt={shot.name}
        sx={{ display: 'block', maxWidth: '100%', maxHeight: '85vh', objectFit: 'contain' }}
      />
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: space.gap,
          px: space.pad,
          py: 1,
          fontSize: 12,
          color: 'text.secondary',
        }}
      >
        <Typography
          noWrap={true}
          sx={{ flex: 1, minWidth: 0, fontSize: 'inherit' }}
          title={shot.name}
        >
          {shot.name}
        </Typography>
        <When value={shot.at} withTime={true} />
        {profile ? (
          <Button size="small" onClick={() => makeCover().catch(reportUnexpected)}>
            {t`Use as profile cover`}
          </Button>
        ) : null}
      </Box>
    </Dialog>
  )
}

export function ScreenshotsTab({ game }: { game: string }) {
  const { t } = useLingui()
  const [gallery, setGallery] = useState<Gallery | null>(null)
  const [error, setError] = useState<unknown>(null)
  const [open, setOpen] = useState<number | null>(null)
  const load = useCallback(() => {
    setError(null)
    Screenshots(game).then(setGallery).catch(setError)
  }, [game])
  useEffect(load, [load])
  const shots = gallery?.shots ?? []
  const sources = gallery?.sources ?? []
  const both = sources.length > 1
  const folderLabel = (source: string) => {
    if (!both) {
      return t`Open folder`
    }
    return source === STEAM ? t`Open Steam's folder` : t`Open the game's folder`
  }
  let body: ReactNode
  if (error !== null) {
    body = (
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
        <Typography sx={{ fontSize: 13, color: 'error.main' }}>{errorMessage(error)}</Typography>
        <Button size="small" onClick={load}>
          {t`Retry`}
        </Button>
      </Box>
    )
  } else if (gallery === null) {
    body = <LoadingRow>{t`Reading your screenshots…`}</LoadingRow>
  } else if (shots.length === 0) {
    body = (
      <EmptyState icon={<Camera size={40} aria-hidden={true} />} title={t`No screenshots yet`}>
        {t`Screenshots you take while playing show here, newest first.`}
      </EmptyState>
    )
  } else {
    body = (
      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: `repeat(auto-fill, minmax(${TILE_MIN_PX}px, 1fr))`,
          gap: space.gap,
        }}
      >
        {shots.map((s, i) => (
          <ButtonBase
            key={`${s.source}/${s.name}`}
            onClick={() => setOpen(i)}
            aria-label={s.name}
            sx={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'stretch',
              borderRadius: '8px',
              overflow: 'hidden',
              bgcolor: 'var(--mortar-paper-78)',
              border: '1px solid var(--mortar-hairline-faint)',
              '&:hover': { borderColor: 'var(--mortar-hairline-16)' },
            }}
          >
            <Box
              component="img"
              src={shotUrl(game, s)}
              alt=""
              loading="lazy"
              sx={{ display: 'block', width: '100%', aspectRatio: '16 / 9', objectFit: 'cover' }}
            />
            <Box
              sx={{
                display: 'flex',
                justifyContent: 'space-between',
                gap: 1,
                px: 1,
                py: 0.5,
                fontSize: 12,
                color: 'text.secondary',
              }}
            >
              <When value={s.at} withTime={true} />
              {both && s.source === STEAM ? <span>{storeName('steam')}</span> : null}
            </Box>
          </ButtonBase>
        ))}
      </Box>
    )
  }
  return (
    <Box sx={{ flex: 1, minHeight: 0, overflow: 'auto', display: 'flex', flexDirection: 'column' }}>
      <PageActions>
        {sources.map((source) => (
          <Button
            key={source}
            variant="outlined"
            color="inherit"
            startIcon={<FolderOpen size={16} />}
            onClick={() => {
              OpenScreenshotsFolder(game, source).catch(reportUnexpected)
            }}
          >
            {folderLabel(source)}
          </Button>
        ))}
      </PageActions>
      <Box
        sx={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          px: space.gutter,
          pt: space.gap,
          pb: space.gutter,
        }}
      >
        {body}
      </Box>
      {open === null ? null : (
        <Viewer
          game={game}
          shots={shots}
          index={open}
          onIndex={setOpen}
          onClose={() => setOpen(null)}
        />
      )}
    </Box>
  )
}
