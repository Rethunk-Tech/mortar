import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  TextField,
  Typography,
} from '@mui/material'
import { useState } from 'react'
import type {
  ModpackForm,
  ModpackOptions,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/models.ts'
import {
  ExportModpackDialog,
  ModpackChangelog,
  ModpackForm as ReadForm,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import { PickImage } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { listNames } from '../i18n/list.ts'
import { ErrorRetry } from '../shell/ErrorRetry.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { useLoaded } from '../shell/useLoaded.ts'
import { space } from '../theme/density.ts'
import { MONO } from '../theme/theme.ts'
import { inlineError, reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'

const DESCRIPTION_MAX = 250
const NAME_PATTERN = /^[A-Za-z0-9_]+$/
const VERSION_PATTERN = /^\d+\.\d+\.\d+$/
const monoField = { '& textarea': { fontFamily: MONO, fontSize: 13 } } as const

const HTTP = /^https?:\/\//

function IconRow({ path, onChange }: { path: string; onChange: (path: string) => void }) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
      <Typography sx={{ flex: 1, fontSize: 13, overflowWrap: 'anywhere' }}>
        {path || t`A plain icon (${{ source: sourceLabel('thunderstore') }} needs a 256x256 PNG)`}
      </Typography>
      <Button
        size="small"
        onClick={() =>
          PickImage(t`Choose icon`)
            .then((picked) => picked && onChange(picked))
            .catch(reportError(t`Could not open the file dialog`))
        }
      >
        {t`Choose icon…`}
      </Button>
      {path ? (
        <Button size="small" onClick={() => onChange('')}>
          {t`Use the plain icon`}
        </Button>
      ) : null}
    </Box>
  )
}

function ConfigFiles({
  all,
  chosen,
  onChange,
}: {
  all: string[]
  chosen: string[]
  onChange: (names: string[]) => void
}) {
  const { t } = useLingui()
  if (all.length === 0) {
    return null
  }
  const files = new Set(chosen)
  return (
    <Box>
      <Typography sx={{ fontSize: 13, fontWeight: 700 }}>
        {t`Config files (${{ count: files.size }} of ${{ total: all.length }})`}
      </Typography>
      <Box sx={{ display: 'flex', gap: space.gap }}>
        <Button size="small" onClick={() => onChange(all)}>
          {t`All`}
        </Button>
        <Button size="small" onClick={() => onChange([])}>
          {t`None`}
        </Button>
      </Box>
      <Box sx={{ maxHeight: 200, overflowY: 'auto' }}>
        {all.map((name) => (
          <FormControlLabel
            key={name}
            sx={{ display: 'flex' }}
            control={
              <Checkbox
                size="small"
                checked={files.has(name)}
                onChange={(e) =>
                  onChange(all.filter((c) => (c === name ? e.target.checked : files.has(c))))
                }
              />
            }
            label={<Typography sx={{ fontSize: 13 }}>{name}</Typography>}
          />
        ))}
      </Box>
    </Box>
  )
}

function Fields({
  game,
  profileId,
  form,
  onClose,
}: {
  game: string
  profileId: string
  form: ModpackForm
  onClose: () => void
}) {
  const { t } = useLingui()
  const [o, setO] = useState<ModpackOptions>(form.options)
  const [busy, run] = usePending()
  const set = <K extends keyof ModpackOptions>(key: K, value: ModpackOptions[K]) =>
    setO((cur) => ({ ...cur, [key]: value }))
  const bad = (() => {
    if (!NAME_PATTERN.test(o.name)) {
      return t`The name may hold only letters, digits and underscores.`
    }
    if (!VERSION_PATTERN.test(o.version)) {
      return t`The version must look like 1.0.0.`
    }
    return o.website !== '' && !HTTP.test(o.website)
      ? t`The website must start with http:// or https://.`
      : ''
  })()
  const regenerate = () =>
    ModpackChangelog(game, profileId, o.version, o.includeDisabled)
      .then((text) => set('changelog', text))
      .catch(reportError(t`Could not write the changelog`))
  const save = () =>
    run(
      () =>
        ExportModpackDialog(game, profileId, o).then((r) => {
          if (!r.path) {
            return
          }
          const left = [...(r.leftOut ?? []), ...(r.disabled ?? [])]
          useToasts.getState().push({
            kind: left.length > 0 ? 'warning' : 'success',
            title: t`Modpack saved`,
            body: left.length > 0 ? t`Left out: ${{ names: listNames(left) }}` : r.path,
          })
          onClose()
        }),
      { errorTitle: t`Could not export the modpack` },
    )
  const left = [...(form.leftOut ?? []), ...(o.includeDisabled ? [] : (form.disabled ?? []))]
  return (
    <>
      <DialogContent dividers={true}>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap, pt: 0.5 }}>
          <Box sx={{ display: 'flex', gap: space.gap }}>
            <TextField
              size="small"
              fullWidth={true}
              label={t`Name`}
              value={o.name}
              onChange={(e) => set('name', e.target.value)}
              error={!NAME_PATTERN.test(o.name)}
            />
            <TextField
              size="small"
              label={t`Version`}
              value={o.version}
              onChange={(e) => set('version', e.target.value)}
              error={!VERSION_PATTERN.test(o.version)}
              helperText={
                form.lastVersion ? t`Last export ${{ lastVersion: form.lastVersion }}` : undefined
              }
              sx={{ width: 150 }}
            />
          </Box>
          <TextField
            size="small"
            label={t`Author`}
            value={o.author}
            onChange={(e) => set('author', e.target.value)}
          />
          <TextField
            size="small"
            label={t`Description`}
            value={o.description}
            onChange={(e) => set('description', e.target.value)}
            slotProps={{ htmlInput: { maxLength: DESCRIPTION_MAX } }}
            helperText={`${[...o.description].length}/${DESCRIPTION_MAX}`}
          />
          <TextField
            size="small"
            label={t`Website`}
            value={o.website}
            onChange={(e) => set('website', e.target.value)}
          />
          <IconRow path={o.iconPath} onChange={(path) => set('iconPath', path)} />
          <TextField
            size="small"
            multiline={true}
            minRows={3}
            maxRows={8}
            label={t`README`}
            placeholder={t`Left empty, a README is written from what the profile holds.`}
            value={o.readme}
            onChange={(e) => set('readme', e.target.value)}
            sx={monoField}
          />
          <TextField
            size="small"
            multiline={true}
            minRows={3}
            maxRows={8}
            label={t`Changelog`}
            value={o.changelog}
            onChange={(e) => set('changelog', e.target.value)}
            sx={monoField}
          />
          <Box>
            <Button size="small" onClick={regenerate}>
              {t`Write the changelog against the last export`}
            </Button>
          </Box>
          <FormControlLabel
            control={
              <Checkbox
                checked={o.includeDisabled}
                onChange={(e) => set('includeDisabled', e.target.checked)}
              />
            }
            label={t`Include disabled mods`}
          />
          <ConfigFiles
            all={form.configs ?? []}
            chosen={o.configs ?? []}
            onChange={(names) => set('configs', names)}
          />
          <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
            {left.length > 0
              ? t`Depends on ${{ dependencies: form.dependencies }} ${{ source: sourceLabel('thunderstore') }} packages. Left out: ${{ names: listNames(left) }}.`
              : t`Depends on ${{ dependencies: form.dependencies }} ${{ source: sourceLabel('thunderstore') }} packages.`}
          </Typography>
          {bad ? <Typography sx={{ fontSize: 13, color: 'error.main' }}>{bad}</Typography> : null}
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Cancel`}</Button>
        <Button variant="contained" disabled={busy || bad !== ''} onClick={save}>
          {t`Export…`}
        </Button>
      </DialogActions>
    </>
  )
}

/** The Thunderstore modpack form: name, version, texts, icon, files and the changelog against the last export. */
export function ModpackDialog({
  game,
  profileId,
  onClose,
}: {
  game: string
  profileId: string
  onClose: () => void
}) {
  const { t } = useLingui()
  const { data, error, reload } = useLoaded<ModpackForm | null>(
    () => ReadForm(game, profileId),
    [game, profileId],
    null,
  )
  return (
    <Dialog open={true} onClose={onClose} fullWidth={true} maxWidth="sm">
      <DialogTitle>{t`Export as ${{ source: sourceLabel('thunderstore') }} modpack`}</DialogTitle>
      {error === null ? null : (
        <DialogContent>
          <ErrorRetry error={inlineError(error)} onRetry={reload} />
        </DialogContent>
      )}
      {error === null && data === null ? (
        <DialogContent>
          <LoadingRow>{t`Reading the profile…`}</LoadingRow>
        </DialogContent>
      ) : null}
      {data ? <Fields game={game} profileId={profileId} form={data} onClose={onClose} /> : null}
    </Dialog>
  )
}
