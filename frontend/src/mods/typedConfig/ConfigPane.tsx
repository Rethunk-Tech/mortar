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
import { CircleHelp, RotateCcw, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import { openProfileOf, useProfiles } from '../../profiles/store.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { SearchField } from '../../shell/SearchField.tsx'
import { space } from '../../theme/density.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { PresetsButton } from '../ConfigPresets.tsx'
import { filterFile, isModified, modifiedCount } from './entries.ts'
import { useTypedConfig } from './store.ts'
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
        <Typography sx={{ overflowWrap: 'anywhere' }}>{entry.label || entry.key}</Typography>
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

// The full-pane editor of one mod's config files, over the Mods tab.
export function ConfigPane() {
  const { t } = useLingui()
  const { mod, files, current, loadError } = useTypedConfig()
  const close = useTypedConfig((s) => s.close)
  const select = useTypedConfig((s) => s.select)
  const target = useTypedConfig((s) => s.target)
  const open = useTypedConfig((s) => s.open)
  const reload = () => {
    if (mod && target) {
      open(mod, target).catch(reportUnexpected)
    }
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
  if (!mod) {
    return null
  }
  const file = files.find((f) => f.name === current)
  const shown = file ? filterFile(file, query) : null
  return (
    <Box
      aria-label={t`Config of ${mod.name}`}
      role="region"
      sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}
    >
      <Box
        sx={{ display: 'flex', alignItems: 'center', gap: space.gap, px: space.pad, py: space.gap }}
      >
        <Typography component="h2" sx={{ fontSize: 18, fontWeight: 600, flex: 1 }}>
          {t`Config of ${mod.name}`}
        </Typography>
        <SearchField
          label={t`Search entries`}
          value={query}
          onChange={setQuery}
          sx={{ width: 260 }}
        />
        {file?.format === 'gmcm' ? null : <PresetsButton mod={mod} />}
        <Button
          variant="outlined"
          size="small"
          disabled={!file || modifiedCount(file) === 0}
          onClick={() => setConfirming(true)}
        >
          {t`Reset all`}
        </Button>
        <IconButton aria-label={t`Close config editor`} onClick={close}>
          <X size={18} />
        </IconButton>
      </Box>
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
            <ListItemButton
              key={f.name}
              selected={f.name === current}
              onClick={() => select(f.name)}
            >
              <ListItemText
                primary={f.format === 'gmcm' ? t`In-game menu` : f.label || f.name}
                slotProps={{ primary: { noWrap: true } }}
              />
            </ListItemButton>
          ))}
        </List>
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
