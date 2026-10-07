import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  List,
  ListItemButton,
  ListItemText,
  Typography,
} from '@mui/material'
import { FolderOpen } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import type { DownloadArchive } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/archivesvc/models.ts'
import {
  DownloadsArchives,
  DownloadsFolders,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/archivesvc/service.ts'
import { formatBytes } from '../i18n/bytes.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { listNames } from '../i18n/list.ts'
import { openSettings } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { ErrorRetry } from '../shell/ErrorRetry.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { useFolderEvent } from '../shell/useFolderEvent.ts'
import { space } from '../theme/density.ts'
import { type InlineError, inlineError, reportError } from '../toasts/report.ts'
import { ArchivePreview } from './ArchivePreview.tsx'
import { listArchives, useDownloadsDialog } from './downloadsDialog.ts'
import { useInstall } from './store.ts'

function useArchives(open: boolean, game: string) {
  const [archives, setArchives] = useState<DownloadArchive[] | null>(null)
  const [error, setError] = useState<InlineError | null>(null)
  const gen = useRef(0)
  const load = useCallback(() => {
    gen.current += 1
    const token = gen.current
    setError(null)
    DownloadsArchives(game)
      .then((found) => token === gen.current && setArchives(found ?? []))
      .catch((e: unknown) => {
        if (token === gen.current) {
          setError(inlineError(e))
        }
      })
  }, [game])
  useEffect(() => {
    if (!(open && game)) {
      return
    }
    setArchives(null)
    setError(null)
    load()
    return () => {
      gen.current += 1
    }
  }, [open, game, load])
  useFolderEvent('library:downloads', game, () => open && load())
  return { archives, error, reload: load }
}

function FolderLine({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useLingui()
  const [dirs, setDirs] = useState<string[]>([])
  useEffect(() => {
    if (open) {
      DownloadsFolders()
        .then((found) => setDirs(found ?? []))
        .catch(() => setDirs([]))
    }
  }, [open])
  if (dirs.length === 0) {
    return null
  }
  return (
    <Box
      sx={{ display: 'flex', alignItems: 'center', gap: space.gap, px: space.pad, pb: space.gap }}
    >
      <Typography
        variant="body2"
        noWrap={true}
        title={dirs.join('\n')}
        sx={{ color: 'text.secondary', minWidth: 0 }}
      >
        {t`Reading ${listNames(dirs)}`}
      </Typography>
      <Button
        size="small"
        sx={{ flexShrink: 0 }}
        onClick={() => {
          onClose()
          openSettings('downloads')
        }}
      >
        {t`Change folder`}
      </Button>
    </Box>
  )
}

function ArchiveRows({
  archives,
  selected,
  onSelect,
}: {
  archives: DownloadArchive[]
  selected: string
  onSelect: (path: string) => void
}) {
  const { t } = useLingui()
  return (
    <List
      dense={true}
      role="listbox"
      aria-label={t`Archives`}
      sx={{ flex: 1, minHeight: 0, overflowY: 'auto' }}
    >
      {archives.map((a) => (
        <ListItemButton
          key={a.path}
          role="option"
          aria-selected={a.path === selected}
          selected={a.path === selected}
          onClick={() => onSelect(a.path)}
        >
          <ListItemText
            primary={a.name}
            secondary={`${formatBytes(a.size)} · ${formatWhen(a.mtime)}`}
            slotProps={{ primary: { noWrap: true, title: a.name } }}
          />
          {a.knownNexus ? <Chip size="small" label={t`From Nexus`} sx={{ ml: 1 }} /> : null}
        </ListItemButton>
      ))}
    </List>
  )
}

function Body({
  archives,
  error,
  onRetry,
  shown,
  query,
  onQuery,
  selected,
  onSelect,
  onClose,
}: {
  archives: DownloadArchive[] | null
  error: InlineError | null
  onRetry: () => void
  shown: DownloadArchive[]
  query: string
  onQuery: (query: string) => void
  selected: string
  onSelect: (path: string) => void
  onClose: () => void
}) {
  const { t } = useLingui()
  if (error) {
    return <ErrorRetry error={error} onRetry={onRetry} />
  }
  if (archives === null) {
    return <LoadingRow>{t`Looking in the downloads folder…`}</LoadingRow>
  }
  if (archives.length === 0) {
    return (
      <EmptyState
        compact={true}
        icon={<FolderOpen size={28} />}
        title={t`No archives to add`}
        action={
          <Button
            onClick={() => {
              onClose()
              openSettings('downloads')
            }}
          >
            {t`Change folder`}
          </Button>
        }
      >
        {t`Every archive in the downloads folder is already in a profile or Mortar's store.`}
      </EmptyState>
    )
  }
  return (
    <>
      <Box
        sx={{ width: 340, flexShrink: 0, display: 'flex', flexDirection: 'column', gap: space.gap }}
      >
        <SearchField label={t`Search archives`} value={query} onChange={onQuery} fullWidth={true} />
        {shown.length === 0 ? (
          <Typography
            sx={{ color: 'text.secondary', px: space.gap }}
          >{t`No archives match`}</Typography>
        ) : (
          <ArchiveRows archives={shown} selected={selected} onSelect={onSelect} />
        )}
      </Box>
      <Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
        {selected ? (
          <ArchivePreview path={selected} />
        ) : (
          <Typography sx={{ color: 'text.secondary', p: space.gap }}>
            {t`Choose an archive to see what is inside`}
          </Typography>
        )}
      </Box>
    </>
  )
}

/** Archives already in the downloads folder that no profile has used: pick one, look inside, add it to the open profile. */
export function DownloadsDialog() {
  const { t } = useLingui()
  const open = useDownloadsDialog((s) => s.open)
  const close = () => useDownloadsDialog.getState().setOpen(false)
  const game = useProfiles((s) => s.game?.id ?? '')
  const hasProfile = useProfiles((s) => s.openId !== '')
  const { archives, error, reload } = useArchives(open, game)
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState('')
  useEffect(() => {
    if (open) {
      setQuery('')
      setSelected('')
    }
  }, [open])
  const shown = listArchives(archives ?? [], query)
  return (
    <Dialog open={open} onClose={close} maxWidth={false}>
      <DialogTitle>{t`From the downloads folder`}</DialogTitle>
      <FolderLine open={open} onClose={close} />
      <DialogContent
        sx={{
          width: 'min(900px, calc(100vw - 96px))',
          height: 460,
          display: 'flex',
          gap: space.pad,
        }}
      >
        <Body
          archives={archives}
          error={error}
          onRetry={reload}
          shown={shown}
          query={query}
          onQuery={setQuery}
          selected={selected}
          onSelect={setSelected}
          onClose={close}
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={close}>{t`Cancel`}</Button>
        <DisabledReason
          title={hasProfile ? t`Choose an archive first.` : t`Open a profile first.`}
          disabled={!(selected && hasProfile)}
        >
          <Button
            variant="contained"
            disabled={!(selected && hasProfile)}
            onClick={() => {
              close()
              useInstall
                .getState()
                .installDownloads([selected])
                .catch(reportError(t`Could not add the archive`))
            }}
          >
            {t`Add to profile`}
          </Button>
        </DisabledReason>
      </DialogActions>
    </Dialog>
  )
}
