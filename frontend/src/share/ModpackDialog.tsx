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
import { Fold } from '../shell/Fold.tsx'
import { SkeletonRows } from '../shell/SkeletonRows.tsx'
import { useLoaded } from '../shell/useLoaded.ts'
import { space } from '../theme/density.ts'
import { MONO } from '../theme/theme.ts'
import { inlineError, reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'

const DESCRIPTION_MAX = 250
const SKELETON_FIELDS = 6
const NAME_PATTERN = /^[A-Za-z0-9_]+$/
const VERSION_PATTERN = /^\d+\.\d+\.\d+$/
const errorText = { fontSize: 'base.fontSize', color: 'error.main' } as const
const monoField = { '& textarea': { fontFamily: MONO, fontSize: 'base.fontSize' } } as const

const HTTP = /^https?:\/\//

function IconRow({ path, onChange }: { path: string; onChange: (path: string) => void }) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, flexWrap: 'wrap' }}>
      <Button
        variant="outlined"
        onClick={() =>
          PickImage(t`Choose icon`)
            .then((picked) => picked && onChange(picked))
            .catch(reportError(t`Could not open the file dialog`))
        }
      >
        {t`Choose icon…`}
      </Button>
      {path ? <Button onClick={() => onChange('')}>{t`Use the plain icon`}</Button> : null}
      <Typography
        sx={{
          flex: 1,
          minWidth: 0,
          fontSize: 'small.fontSize',
          color: 'text.secondary',
          overflowWrap: 'anywhere',
        }}
      >
        {path || t`A plain icon (${{ source: sourceLabel('thunderstore') }} needs a 256x256 PNG)`}
      </Typography>
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
  // A profile's configs run to a hundred or more, so the list stays folded until wanted.
  return (
    <Fold title={t`Config files (${{ count: files.size }} of ${{ total: all.length }})`}>
      <Box sx={{ display: 'flex', gap: space.gap }}>
        <Button size="small" variant="outlined" onClick={() => onChange(all)}>
          {t`All`}
        </Button>
        <Button size="small" variant="outlined" onClick={() => onChange([])}>
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
            label={<Typography sx={{ fontSize: 'base.fontSize' }}>{name}</Typography>}
          />
        ))}
      </Box>
    </Fold>
  )
}

const section = { fontSize: 'base.fontSize', fontWeight: 700, mt: 1 } as const

// The package's README and changelog, and the button that writes the changelog from the last export.
function Texts({
  readme,
  changelog,
  onReadme,
  onChangelog,
  onRegenerate,
}: {
  readme: string
  changelog: string
  onReadme: (text: string) => void
  onChangelog: (text: string) => void
  onRegenerate: () => void
}) {
  const { t } = useLingui()
  return (
    <>
      <Typography sx={section}>{t`Texts`}</Typography>
      <TextField
        size="small"
        multiline={true}
        minRows={3}
        maxRows={8}
        label={t`README`}
        placeholder={t`Left empty, a README is written from what the profile holds.`}
        value={readme}
        onChange={(e) => onReadme(e.target.value)}
        sx={monoField}
      />
      <TextField
        size="small"
        multiline={true}
        minRows={3}
        maxRows={8}
        label={t`Changelog`}
        value={changelog}
        onChange={(e) => onChangelog(e.target.value)}
        sx={monoField}
      />
      <Button
        size="small"
        variant="outlined"
        onClick={onRegenerate}
        sx={{ alignSelf: 'flex-start' }}
      >
        {t`Write the changelog against the last export`}
      </Button>
    </>
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
          <Typography sx={{ ...section, mt: 0 }}>{t`Package`}</Typography>
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
          <Texts
            readme={o.readme}
            changelog={o.changelog}
            onReadme={(v) => set('readme', v)}
            onChangelog={(v) => set('changelog', v)}
            onRegenerate={regenerate}
          />
          <Typography sx={section}>{t`Contents`}</Typography>
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
          <Typography sx={{ fontSize: 'base.fontSize', color: 'text.secondary' }}>
            {left.length > 0
              ? t`Depends on ${{ dependencies: form.dependencies }} ${{ source: sourceLabel('thunderstore') }} packages. Left out: ${{ names: listNames(left) }}.`
              : t`Depends on ${{ dependencies: form.dependencies }} ${{ source: sourceLabel('thunderstore') }} packages.`}
          </Typography>
          {bad ? <Typography sx={errorText}>{bad}</Typography> : null}
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
          <SkeletonRows label={t`Reading the profile…`} count={SKELETON_FIELDS} height={40} />
        </DialogContent>
      ) : null}
      {data ? <Fields game={game} profileId={profileId} form={data} onClose={onClose} /> : null}
    </Dialog>
  )
}
