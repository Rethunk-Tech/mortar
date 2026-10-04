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
} from '@mui/material'
import { FolderOpen } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { DownloadArchive } from '../../bindings/github.com/Rethunk-AI/mortar/internal/archivesvc/models.ts'
import { DownloadsArchives } from '../../bindings/github.com/Rethunk-AI/mortar/internal/archivesvc/service.ts'
import { formatBytes } from '../i18n/bytes.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { useProfiles } from '../profiles/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { reportError } from '../toasts/report.ts'
import { ArchivePreview } from './ArchivePreview.tsx'
import { listArchives, useDownloadsDialog } from './downloadsDialog.ts'
import { useInstall } from './store.ts'

function useArchives(open: boolean, game: string) {
  const { t } = useLingui()
  const [archives, setArchives] = useState<DownloadArchive[] | null>(null)
  useEffect(() => {
    if (!(open && game)) {
      return
    }
    let active = true
    setArchives(null)
    DownloadsArchives(game)
      .then((found) => active && setArchives(found ?? []))
      .catch(reportError(t`Could not read the downloads folder`))
    return () => {
      active = false
    }
  }, [open, game, t])
  return archives
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
    <List dense={true} sx={{ flex: 1, minHeight: 0, overflowY: 'auto' }}>
      {archives.map((a) => (
        <ListItemButton
          key={a.path}
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
  shown,
  query,
  onQuery,
  selected,
  onSelect,
}: {
  archives: DownloadArchive[] | null
  shown: DownloadArchive[]
  query: string
  onQuery: (query: string) => void
  selected: string
  onSelect: (path: string) => void
}) {
  const { t } = useLingui()
  if (archives === null) {
    return <LoadingRow>{t`Looking in the downloads folder…`}</LoadingRow>
  }
  if (archives.length === 0) {
    return (
      <EmptyState compact={true} icon={<FolderOpen size={28} />} title={t`No archives to add`}>
        {t`Every archive in the downloads folder is already in a profile or Mortar's store.`}
      </EmptyState>
    )
  }
  return (
    <>
      <Box sx={{ width: 340, flexShrink: 0, display: 'flex', flexDirection: 'column', gap: 1 }}>
        <SearchField label={t`Search archives`} value={query} onChange={onQuery} fullWidth={true} />
        <ArchiveRows archives={shown} selected={selected} onSelect={onSelect} />
      </Box>
      <Box sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}>
        {selected ? <ArchivePreview path={selected} /> : null}
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
  const archives = useArchives(open, game)
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
      <DialogContent
        sx={{ width: 'min(900px, calc(100vw - 96px))', height: 460, display: 'flex', gap: 2 }}
      >
        <Body
          archives={archives}
          shown={shown}
          query={query}
          onQuery={setQuery}
          selected={selected}
          onSelect={setSelected}
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
                .install([selected])
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
