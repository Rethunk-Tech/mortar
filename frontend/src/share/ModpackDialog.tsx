import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, FormControlLabel, TextField, Typography } from '@mui/material'
import { useId, useState } from 'react'
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
import { AppDialog, DialogFooter } from '../shell/dialog/AppDialog.tsx'
import { Field, FieldRow, FormDialog, FormSection, TextInput } from '../shell/dialog/FormDialog.tsx'
import { ErrorRetry } from '../shell/ErrorRetry.tsx'
import { Fold } from '../shell/Fold.tsx'
import { SkeletonRows } from '../shell/SkeletonRows.tsx'
import { sameValue } from '../shell/sameValue.ts'
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

type SetOption = <K extends keyof ModpackOptions>(key: K, value: ModpackOptions[K]) => void

// A tall monospace text box in a Field: label above, never in the border.
function TextBox({
  label,
  value,
  onChange,
  placeholder,
}: {
  label: string
  value: string
  onChange: (text: string) => void
  placeholder?: string
}) {
  const id = useId()
  return (
    <Field label={label} htmlFor={id}>
      <TextField
        id={id}
        size="small"
        fullWidth={true}
        multiline={true}
        minRows={3}
        maxRows={8}
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        sx={monoField}
      />
    </Field>
  )
}

function PackageSection({
  o,
  set,
  lastVersion,
}: {
  o: ModpackOptions
  set: SetOption
  lastVersion: string
}) {
  const { t } = useLingui()
  return (
    <FormSection title={t`Package`}>
      <FieldRow>
        <TextInput
          label={t`Name`}
          value={o.name}
          onChange={(v) => set('name', v)}
          error={NAME_PATTERN.test(o.name) ? '' : t`Letters, digits and underscores only.`}
        />
        <TextInput
          label={t`Version`}
          value={o.version}
          onChange={(v) => set('version', v)}
          error={VERSION_PATTERN.test(o.version) ? '' : t`Like 1.0.0.`}
          hint={lastVersion ? t`Last export ${{ lastVersion }}` : undefined}
        />
      </FieldRow>
      <TextInput label={t`Author`} value={o.author} onChange={(v) => set('author', v)} />
      <TextInput
        label={t`Description`}
        value={o.description}
        onChange={(v) => set('description', v)}
        maxLength={DESCRIPTION_MAX}
      />
      <TextInput label={t`Website`} value={o.website} onChange={(v) => set('website', v)} />
      <IconRow path={o.iconPath} onChange={(path) => set('iconPath', path)} />
    </FormSection>
  )
}

// The package's README and changelog, and the button that writes the changelog from the last export.
function TextsSection({
  o,
  set,
  onRegenerate,
}: {
  o: ModpackOptions
  set: SetOption
  onRegenerate: () => void
}) {
  const { t } = useLingui()
  return (
    <FormSection title={t`Texts`}>
      <TextBox
        label={t`README`}
        placeholder={t`Left empty, a README is written from what the profile holds.`}
        value={o.readme}
        onChange={(v) => set('readme', v)}
      />
      <TextBox label={t`Changelog`} value={o.changelog} onChange={(v) => set('changelog', v)} />
      <Button
        size="small"
        variant="outlined"
        onClick={onRegenerate}
        sx={{ alignSelf: 'flex-start' }}
      >
        {t`Write the changelog against the last export`}
      </Button>
    </FormSection>
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
  const set: SetOption = (key, value) => setO((cur) => ({ ...cur, [key]: value }))
  let issue = ''
  if (!NAME_PATTERN.test(o.name)) {
    issue = t`The name may hold only letters, digits and underscores.`
  } else if (!VERSION_PATTERN.test(o.version)) {
    issue = t`The version must look like 1.0.0.`
  } else if (o.website !== '' && !HTTP.test(o.website)) {
    issue = t`The website must start with http:// or https://.`
  }
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
    <FormDialog
      open={true}
      onClose={onClose}
      title={t`Export as ${{ source: sourceLabel('thunderstore') }} modpack`}
      submitLabel={t`Export…`}
      issue={issue}
      dirty={!sameValue(o, form.options)}
      busy={busy}
      onSubmit={save}
    >
      <PackageSection o={o} set={set} lastVersion={form.lastVersion} />
      <TextsSection o={o} set={set} onRegenerate={regenerate} />
      <FormSection title={t`Contents`}>
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
      </FormSection>
    </FormDialog>
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
  if (data) {
    return <Fields game={game} profileId={profileId} form={data} onClose={onClose} />
  }
  return (
    <AppDialog
      open={true}
      onClose={onClose}
      title={t`Export as ${{ source: sourceLabel('thunderstore') }} modpack`}
      footer={
        <DialogFooter>
          <Button onClick={onClose}>{t`Cancel`}</Button>
        </DialogFooter>
      }
    >
      {error === null ? (
        <SkeletonRows label={t`Reading the profile…`} count={SKELETON_FIELDS} height={40} />
      ) : (
        <ErrorRetry error={inlineError(error)} onRetry={reload} />
      )}
    </AppDialog>
  )
}
