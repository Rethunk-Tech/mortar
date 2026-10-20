import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, IconButton, MenuItem, TextField, Typography } from '@mui/material'
import { CircleHelp, RotateCcw } from 'lucide-react'
import { type RefObject, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react'
import type { Mod } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { openProfileOf, useProfiles } from '../../profiles/store.ts'
import { useSettings } from '../../settings/store.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { SearchField } from '../../shell/SearchField.tsx'
import { Tooltip } from '../../shell/Tooltip.tsx'
import { OnlyHereFileButton } from '../../sync/pins.tsx'
import { space } from '../../theme/density.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { PresetsButton } from '../ConfigPresets.tsx'
import { localId } from '../dependents.ts'
import { configApi } from './api.ts'
import { showsOneFile, useConfigList } from './configList.ts'
import { filterFile, humanizeKey, isModified, modifiedCount } from './entries.ts'
import { SectionNav } from './SectionNav.tsx'
import { type Target, useTypedConfig } from './store.ts'
import type { ConfigEntry } from './types.ts'
import { EntryWidget } from './Widgets.tsx'

const FILE_PICKER_WIDTH_PX = 260

function EntryRow({ section, entry }: { section: string; entry: ConfigEntry }) {
  const { t } = useLingui()
  const set = useTypedConfig((s) => s.set)
  const reset = useTypedConfig((s) => s.reset)
  const error = useTypedConfig((s) => s.errors[`${s.current}/${section}/${entry.key}`] ?? '')
  return (
    <Box
      className="entry-row"
      sx={{
        display: 'grid',
        gridTemplateColumns: 'minmax(0, 1fr) auto 28px',
        alignItems: 'center',
        columnGap: 1.5,
        minHeight: space.row,
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
          <Tooltip title={entry.description} describeChild={true}>
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
  target,
  file,
  files,
  mod,
  single,
  query,
  onQuery,
  onSelect,
  onReset,
  onDelete,
}: {
  target: Target
  file: PaneFile | undefined
  files: PaneFile[]
  mod: Mod | null
  single: boolean
  query: string
  onQuery: (query: string) => void
  onSelect: (name: string) => void
  onReset: () => void
  onDelete?: (() => void) | undefined
}) {
  const { t } = useLingui()
  const labelOf = (f: PaneFile) => (f.format === 'gmcm' ? t`In-game menu` : f.label || f.name)
  const labels = files.map(labelOf)
  // Two files labelled alike (one plugin's .cfg and .json) are told apart by their file names.
  const fileLabel = (f: PaneFile) =>
    labels.filter((l) => l === labelOf(f)).length > 1 ? f.name : labelOf(f)
  // One file is named; several are picked from a dropdown in the same place, so the editor keeps the width.
  return (
    <Box
      sx={{ display: 'flex', alignItems: 'center', gap: space.gap, px: space.pad, py: space.gap }}
    >
      <Box sx={{ flex: 1, minWidth: 0 }}>
        {single ? (
          <Typography
            noWrap={true}
            title={file?.name ?? ''}
            sx={{ fontSize: 12, color: 'text.secondary' }}
          >
            {file ? fileLabel(file) : ''}
          </Typography>
        ) : (
          <TextField
            select={true}
            size="small"
            value={file?.name ?? ''}
            onChange={(e) => onSelect(e.target.value)}
            slotProps={{ htmlInput: { 'aria-label': t`Config file` } }}
            sx={{ minWidth: FILE_PICKER_WIDTH_PX, maxWidth: '100%' }}
          >
            {files.map((f) => (
              <MenuItem key={f.name} value={f.name} title={f.name}>
                {fileLabel(f)}
              </MenuItem>
            ))}
          </TextField>
        )}
      </Box>
      {onDelete ? (
        <Button variant="outlined" color="error" size="small" onClick={onDelete}>
          {t`Delete file`}
        </Button>
      ) : null}
      {file?.format === 'text' ? null : (
        <>
          <SearchField
            label={t`Search entries`}
            value={query}
            onChange={onQuery}
            sx={{ width: 260 }}
          />
          {file?.format === 'gmcm' || file?.format === 'json' || !mod ? null : (
            <PresetsButton mod={mod} />
          )}
          {file && file.format !== 'gmcm' && file.format !== 'text' ? (
            <OnlyHereFileButton
              game={target.game}
              profile={target.profile}
              file={file.format === 'smapi' ? `${localId(target.id)}/${file.name}` : file.name}
            />
          ) : null}
          <Button
            variant="outlined"
            size="small"
            disabled={!file || modifiedCount(file) === 0}
            onClick={onReset}
          >
            {t`Reset all`}
          </Button>
        </>
      )}
    </Box>
  )
}

// A file this long gets the list of its sections beside the entries.
const NAV_MIN_SECTIONS = 6
const NAV_MIN_SCREENS = 2
// On a window this wide, centred controls sit mid-row instead of at the far right.
const CENTRED_FROM = '@media (min-width: 1440px)'

// Whether the file is long enough for the section list: many sections, or more than two screens of entries.
function useLongFile(list: RefObject<HTMLElement | null>, sections: number, entries: number) {
  const [long, setLong] = useState(false)
  useLayoutEffect(() => {
    const el = list.current
    setLong(
      sections >= NAV_MIN_SECTIONS ||
        (entries > 0 && el !== null && el.scrollHeight > NAV_MIN_SCREENS * el.clientHeight),
    )
  }, [list, sections, entries])
  return long
}

function Sections({ shown }: { shown: ReturnType<typeof filterFile> | null }) {
  const { t } = useLingui()
  const list = useRef<HTMLElement | null>(null)
  const centred = useSettings((s) => s.centerConfigControls)
  const names = useMemo(
    () => (shown?.sections ?? []).map((s) => s.name).filter((n) => n !== ''),
    [shown],
  )
  const entries = (shown?.sections ?? []).reduce((n, s) => n + s.entries.length, 0)
  const long = useLongFile(list, names.length, entries)
  return (
    <>
      <Box
        ref={list}
        sx={{
          flex: 1,
          minWidth: 0,
          overflowY: 'auto',
          p: space.pad,
          display: 'flex',
          flexDirection: 'column',
          gap: space.gap,
          ...(centred ? { [CENTRED_FROM]: { '& .entry-row': { pr: '45%' } } } : {}),
        }}
      >
        {shown?.sections.map((section) => (
          <Box
            key={section.name}
            component="section"
            data-section={section.name}
            sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}
          >
            {section.name ? (
              <Typography component="h2" sx={{ fontSize: 16, fontWeight: 600, mt: 1 }}>
                {section.name}
              </Typography>
            ) : null}
            {section.entries.map((entry) => (
              <EntryRow key={entry.key} section={section.name} entry={entry} />
            ))}
          </Box>
        ))}
        {shown && shown.sections.length === 0 ? (
          <Typography sx={{ color: 'text.secondary' }}>{t`No entries match.`}</Typography>
        ) : null}
      </Box>
      {long && names.length > 1 ? <SectionNav list={list} names={names} /> : null}
    </>
  )
}

// A text file (YAML, TOML, XML, INI, JSON or plain text a mod keeps beside its .cfg) has no schema, so it is edited
// whole, and saved only when asked.
function TextFile({ target, name }: { target: Target; name: string }) {
  const { t } = useLingui()
  const [saved, setSaved] = useState<string | null>(null)
  const [text, setText] = useState('')
  const [error, setError] = useState('')
  const { game, profile, key, id } = target
  useEffect(() => {
    const at = { game, profile, key, id }
    let live = true
    setSaved(null)
    setError('')
    configApi
      .text(at, name)
      .then((v) => {
        if (live) {
          setSaved(v)
          setText(v)
        }
      })
      .catch((e: unknown) => live && setError(String(e)))
    return () => {
      live = false
    }
  }, [game, profile, key, id, name])
  const save = () => {
    setError('')
    configApi
      .setText(target, name, text)
      .then(() => setSaved(text))
      .catch((e: unknown) => setError(String(e)))
  }
  const dirty = saved !== null && text !== saved
  return (
    <Box
      sx={{
        flex: 1,
        minWidth: 0,
        minHeight: 0,
        display: 'flex',
        flexDirection: 'column',
        gap: space.gap,
        p: space.pad,
      }}
    >
      {error ? (
        <Typography role="alert" sx={{ fontSize: 12, color: 'error.main' }}>
          {saved === null ? t`Could not open: ${error}` : t`Could not save: ${error}`}
        </Typography>
      ) : null}
      {/* MUI sizes a multiline textarea inline to its rows; it is forced to fill the pane instead and scrolls. */}
      <TextField
        multiline={true}
        rows={1}
        fullWidth={true}
        disabled={saved === null}
        value={text}
        onChange={(e) => setText(e.target.value)}
        slotProps={{
          htmlInput: { 'aria-label': t`Contents of ${name}`, spellCheck: false },
        }}
        sx={{
          flex: 1,
          minHeight: 0,
          '& .MuiInputBase-root': { height: '100%', alignItems: 'stretch' },
          '& textarea': {
            height: '100% !important',
            overflow: 'auto',
            fontFamily: 'monospace',
            fontSize: 13,
          },
        }}
      />
      <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: space.gap }}>
        <Button
          variant="text"
          size="small"
          sx={{ mr: 'auto' }}
          disabled={saved === null}
          onClick={() =>
            configApi.openText(target, name).catch((e: unknown) => setError(String(e)))
          }
        >
          {t`Open in editor`}
        </Button>
        <Button
          variant="outlined"
          size="small"
          disabled={!dirty}
          onClick={() => setText(saved ?? '')}
        >
          {t`Revert`}
        </Button>
        <Button variant="contained" size="small" disabled={!dirty} onClick={save}>
          {t`Save`}
        </Button>
      </Box>
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
  landing,
}: {
  name: string
  mod: Mod | null
  target: Target
  file?: string
  /** The file and search to open on, from a setting found by the profile-wide search. */
  landing?: { file: string; query: string }
}) {
  const { t } = useLingui()
  const files = useTypedConfig((s) => s.files)
  const current = useTypedConfig((s) => s.current)
  const loadError = useTypedConfig((s) => s.loadError)
  const select = useTypedConfig((s) => s.select)
  const open = useTypedConfig((s) => s.open)
  const { game, profile, key, id } = target
  useEffect(() => {
    open({ game, profile, key, id }, landing?.file ?? initialFile).catch(reportUnexpected)
  }, [open, game, profile, key, id, initialFile, landing?.file])
  const reload = () => {
    open(target, initialFile).catch(reportUnexpected)
  }
  const resetAll = useTypedConfig((s) => s.resetAll)
  const [query, setQuery] = useState(landing?.query ?? '')
  const [confirming, setConfirming] = useState(false)
  const [deleting, setDeleting] = useState(false)
  // A preset applied from here rewrites the file, so the profile's change time rereads it.
  const updated = useProfiles((st) => openProfileOf(st)?.updated)
  const seenUpdated = useRef(updated)
  useEffect(() => {
    if (updated === seenUpdated.current) {
      return
    }
    seenUpdated.current = updated
    const shown = useTypedConfig.getState().current
    if (shown !== '') {
      select(shown).catch(() => undefined)
    }
  }, [updated, select])
  const file = files.find((f) => f.name === current)
  const single = showsOneFile(files.length, initialFile)
  return (
    <Box
      aria-label={t`Config of ${name}`}
      role="region"
      sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}
    >
      <Toolbar
        target={target}
        file={file}
        files={files}
        mod={mod}
        single={single}
        query={query}
        onQuery={setQuery}
        onSelect={(n) => {
          select(n).catch(() => undefined)
        }}
        onReset={() => setConfirming(true)}
        onDelete={!mod && initialFile ? () => setDeleting(true) : undefined}
      />
      {loadError ? (
        <Alert
          variant="compact"
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
        {file?.format === 'text' ? (
          <TextFile key={file.name} target={target} name={file.name} />
        ) : (
          <Sections shown={file ? filterFile(file, query) : null} />
        )}
      </Box>
      <ConfirmDialog
        open={deleting}
        title={t`Delete ${initialFile ?? name}?`}
        body={t`No mod owns this file. It is removed from this profile, and the game makes a new one if it needs it.`}
        confirmLabel={t`Delete`}
        color="error"
        onCancel={() => setDeleting(false)}
        onConfirm={() => {
          setDeleting(false)
          configApi
            .deleteUnowned(target, name)
            .then(() => {
              useConfigList.getState().select(profile, '')
              return useConfigList.getState().load(game, profile, '', true)
            })
            .catch(reportUnexpected)
        }}
      />
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
