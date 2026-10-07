import { useLingui } from '@lingui/react/macro'
import {
  Alert,
  Box,
  Button,
  IconButton,
  List,
  ListItemButton,
  ListItemText,
  Tooltip,
  Typography,
} from '@mui/material'
import { CircleHelp, RotateCcw } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Mod } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { openProfileOf, useProfiles } from '../../profiles/store.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { SearchField } from '../../shell/SearchField.tsx'
import { space } from '../../theme/density.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { PresetsButton } from '../ConfigPresets.tsx'
import { filterFile, humanizeKey, isModified, modifiedCount } from './entries.ts'
import { type Target, useTypedConfig } from './store.ts'
import type { ConfigEntry } from './types.ts'
import { EntryWidget } from './Widgets.tsx'

const FILES_WIDTH_PX = 220

function EntryRow({ section, entry }: { section: string; entry: ConfigEntry }) {
  const { t } = useLingui()
  const set = useTypedConfig((s) => s.set)
  const reset = useTypedConfig((s) => s.reset)
  const error = useTypedConfig((s) => s.errors[`${s.current}/${section}/${entry.key}`] ?? '')
  return (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: 'minmax(0, 1fr) auto 28px',
        alignItems: 'center',
        columnGap: 1.5,
        minHeight: 48,
        px: space.pad,
        py: 0.5,
        borderRadius: '6px',
        bgcolor: 'var(--mortar-overlay-30)',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, minWidth: 0 }}>
        <Typography title={entry.key} sx={{ overflowWrap: 'normal', wordBreak: 'normal' }}>
          {entry.label || humanizeKey(entry.key)}
        </Typography>
        {entry.description ? (
          <Tooltip title={entry.description}>
            <IconButton
              size="small"
              aria-label={t`About ${entry.key}`}
              sx={{ color: 'text.secondary' }}
            >
              <CircleHelp size={14} />
            </IconButton>
          </Tooltip>
        ) : null}
      </Box>
      {entry.readOnly ? (
        <Typography
          sx={{ fontSize: 13, color: 'text.secondary' }}
        >{t`Change it in the game`}</Typography>
      ) : (
        <EntryWidget
          entry={entry}
          label={entry.label || entry.key}
          onChange={(v) => set(section, entry.key, v)}
        />
      )}
      {isModified(entry) ? (
        <Tooltip title={entry.pending ? t`Keep the game's current value` : t`Reset to default`}>
          <IconButton
            size="small"
            aria-label={
              entry.pending
                ? t`Discard the change to ${{ name: entry.label || entry.key }}`
                : t`Reset ${entry.key} to its default`
            }
            onClick={() => reset(section, entry.key)}
          >
            <RotateCcw size={14} />
          </IconButton>
        </Tooltip>
      ) : (
        <span />
      )}
      {entry.pending ? (
        <Typography sx={{ gridColumn: '1 / -1', fontSize: 12, color: 'text.secondary' }}>
          {t`Applied when the game next starts`}
        </Typography>
      ) : null}
      {entry.note ? (
        <Typography sx={{ gridColumn: '1 / -1', fontSize: 12, color: 'warning.main' }}>
          {t`The game could not apply the last change: ${{ reason: entry.note }}`}
        </Typography>
      ) : null}
      {error ? (
        <Typography role="alert" sx={{ gridColumn: '1 / -1', fontSize: 12, color: 'error.main' }}>
          {t`Could not save: ${error}`}
        </Typography>
      ) : null}
    </Box>
  )
}

type PaneFile = ReturnType<typeof useTypedConfig.getState>['files'][number]

function Toolbar({
  file,
  mod,
  single,
  query,
  onQuery,
  onReset,
}: {
  file: PaneFile | undefined
  mod: Mod | null
  single: boolean
  query: string
  onQuery: (query: string) => void
  onReset: () => void
}) {
  const { t } = useLingui()
  // One file needs no picker: the editor takes the width and the toolbar names the file.
  let fileCaption = ''
  if (single && file) {
    fileCaption = file.format === 'gmcm' ? t`In-game menu` : file.name
  }
  return (
    <Box
      sx={{ display: 'flex', alignItems: 'center', gap: space.gap, px: space.pad, py: space.gap }}
    >
      <Typography
        noWrap={true}
        title={file?.name ?? ''}
        sx={{ flex: 1, minWidth: 0, fontSize: 12, color: 'text.secondary' }}
      >
        {fileCaption}
      </Typography>
      <SearchField label={t`Search entries`} value={query} onChange={onQuery} sx={{ width: 260 }} />
      {file?.format === 'gmcm' || !mod ? null : <PresetsButton mod={mod} />}
      <Button
        variant="outlined"
        size="small"
        disabled={!file || modifiedCount(file) === 0}
        onClick={onReset}
      >
        {t`Reset all`}
      </Button>
    </Box>
  )
}

function FileList({
  files,
  current,
  onSelect,
}: {
  files: PaneFile[]
  current: string
  onSelect: (name: string) => void
}) {
  const { t } = useLingui()
  return (
    <List
      aria-label={t`Config files`}
      sx={{
        width: FILES_WIDTH_PX,
        flexShrink: 0,
        overflowY: 'auto',
        borderRight: '1px solid var(--mortar-hairline)',
      }}
    >
      {files.map((f) => (
        <ListItemButton key={f.name} selected={f.name === current} onClick={() => onSelect(f.name)}>
          <ListItemText
            primary={f.format === 'gmcm' ? t`In-game menu` : f.label || f.name}
            slotProps={{ primary: { noWrap: true } }}
          />
        </ListItemButton>
      ))}
    </List>
  )
}

function Sections({ shown }: { shown: ReturnType<typeof filterFile> | null }) {
  const { t } = useLingui()
  return (
    <Box
      sx={{
        flex: 1,
        minWidth: 0,
        overflowY: 'auto',
        p: space.pad,
        display: 'flex',
        flexDirection: 'column',
        gap: space.gap,
      }}
    >
      {shown?.sections.map((section) => (
        <Box
          key={section.name}
          component="section"
          sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}
        >
          <Typography component="h3" sx={{ fontSize: 16, fontWeight: 600, mt: 1 }}>
            {section.name}
          </Typography>
          {section.entries.map((entry) => (
            <EntryRow key={entry.key} section={section.name} entry={entry} />
          ))}
        </Box>
      ))}
      {shown && shown.sections.length === 0 ? (
        <Typography sx={{ color: 'text.secondary' }}>{t`No entries match.`}</Typography>
      ) : null}
    </Box>
  )
}

// The editor of one selection's config files, beside the Config page's list. A selection without a mod (the unowned
// .cfg files) opens on `file` and offers no presets.
export function ConfigPane({
  name,
  mod,
  target,
  file: initialFile,
}: {
  name: string
  mod: Mod | null
  target: Target
  file?: string
}) {
  const { t } = useLingui()
  const { files, current, loadError } = useTypedConfig()
  const select = useTypedConfig((s) => s.select)
  const open = useTypedConfig((s) => s.open)
  const { game, profile, key, id } = target
  useEffect(() => {
    open({ game, profile, key, id }, initialFile).catch(reportUnexpected)
  }, [open, game, profile, key, id, initialFile])
  const reload = () => {
    open(target, initialFile).catch(reportUnexpected)
  }
  const resetAll = useTypedConfig((s) => s.resetAll)
  const [query, setQuery] = useState('')
  const [confirming, setConfirming] = useState(false)
  // A preset applied from here rewrites the file, so the profile's change time rereads it.
  const updated = useProfiles((st) => openProfileOf(st)?.updated)
  useEffect(() => {
    if (updated !== undefined && current !== '') {
      select(current).catch(() => undefined)
    }
  }, [updated, current, select])
  const file = files.find((f) => f.name === current)
  const single = files.length <= 1
  return (
    <Box
      aria-label={t`Config of ${name}`}
      role="region"
      sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}
    >
      <Toolbar
        file={file}
        mod={mod}
        single={single}
        query={query}
        onQuery={setQuery}
        onReset={() => setConfirming(true)}
      />
      {loadError ? (
        <Alert
          severity="error"
          sx={{ mx: space.gutter }}
          action={
            <Button color="inherit" size="small" onClick={reload}>
              {t`Retry`}
            </Button>
          }
        >
          {loadError}
        </Alert>
      ) : null}
      <Box sx={{ flex: 1, minHeight: 0, display: 'flex' }}>
        {single ? null : <FileList files={files} current={current} onSelect={(n) => select(n)} />}
        <Sections shown={file ? filterFile(file, query) : null} />
      </Box>
      <ConfirmDialog
        open={confirming}
        title={t`Reset all entries?`}
        body={
          file?.format === 'gmcm'
            ? t`Every change waiting for the next start is discarded.`
            : t`Every entry in ${current} goes back to its default.`
        }
        confirmLabel={t`Reset all`}
        color="warning"
        onCancel={() => setConfirming(false)}
        onConfirm={() => {
          setConfirming(false)
          resetAll().catch(() => undefined)
        }}
      />
    </Box>
  )
}
