import { useLingui } from '@lingui/react/macro'
import { Box, Button, MenuItem, TextField, Typography } from '@mui/material'
import { useId, useState } from 'react'
import type { ServerSetting } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/models.ts'
import {
  MirrorClient,
  ServerSettings as ReadForm,
  SetServerSetting,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import { Field, FormSection, SelectField } from '../shell/dialog/FormDialog.tsx'
import { useLoaded } from '../shell/useLoaded.ts'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { listProfiles } from './listProfiles.ts'
import { serverFieldChanged, serverFieldError } from './serverForm.ts'

function ServerField({
  setting,
  onSave,
}: {
  setting: ServerSetting
  onSave: (value: string) => void
}) {
  const { t } = useLingui()
  const [draft, setDraft] = useState(setting.value)
  const id = useId()
  const error = serverFieldError(setting, draft)
  const commit = () => {
    if (!error && serverFieldChanged(setting, draft)) {
      onSave(draft)
      if (setting.secret) {
        setDraft('')
      }
    }
  }
  const errors = {
    required: t`Required before the server can start.`,
    invalid: t`This value is not allowed.`,
  }
  const message = error ? errors[error] : setting.hint
  return (
    <Field
      label={setting.name}
      htmlFor={id}
      hint={error ? undefined : message}
      error={error ? message : undefined}
    >
      <TextField
        id={id}
        fullWidth={true}
        size="small"
        type={setting.secret ? 'password' : 'text'}
        autoComplete="off"
        value={draft}
        placeholder={
          setting.secret && setting.set
            ? t`Saved. Type to change it.`
            : setting.default || undefined
        }
        error={Boolean(error)}
        onChange={(event) => setDraft(event.target.value)}
        onBlur={commit}
        onKeyDown={(event) => event.key === 'Enter' && commit()}
      />
    </Field>
  )
}

const NO_CLIENTS: { id: string; name: string }[] = []

// The launch settings of a dedicated server profile (name, port, world, password), and a picker that fills the profile
// with the mods of one of the client game's profiles. Nothing shows for a game that is not a server.
export function ServerSettings({ gameId, profileId }: { gameId: string; profileId: string }) {
  const { t } = useLingui()
  const { data: form, reload } = useLoaded(
    () => ReadForm(gameId, profileId),
    [gameId, profileId],
    null,
    reportUnexpected,
  )
  const [from, setFrom] = useState('')
  const [queued, setQueued] = useState<number | null>(null)
  const clientGame = form?.clients ?? ''
  const { data: clients } = useLoaded(
    clientGame
      ? () =>
          listProfiles(clientGame).then((all) =>
            (all ?? []).filter((p) => !p.error).map((p) => ({ id: p.id, name: p.name })),
          )
      : null,
    [clientGame],
    NO_CLIENTS,
    reportUnexpected,
  )
  if (!form?.clients) {
    return null
  }
  const save = (id: string, value: string) =>
    SetServerSetting(gameId, profileId, id, value).then(reload).catch(reportUnexpected)
  return (
    <FormSection
      title={t`Server`}
      description={t`Applied the next time the server starts. The password stays in this profile's folder.`}
    >
      {(form.settings ?? []).map((s) => (
        <ServerField
          key={`${s.id}:${s.value}:${s.set}`}
          setting={s}
          onSave={(v) => save(s.id, v)}
        />
      ))}
      <Box sx={{ display: 'flex', gap: space.gap, alignItems: 'flex-end' }}>
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <SelectField label={t`Use mods from…`} value={from} onChange={setFrom}>
            {clients.map((c) => (
              <MenuItem key={c.id} value={c.id}>
                {c.name}
              </MenuItem>
            ))}
          </SelectField>
        </Box>
        <Button
          disabled={from === ''}
          onClick={() =>
            MirrorClient(gameId, profileId, from)
              .then((r) => setQueued(r.queued))
              .catch(reportUnexpected)
          }
        >
          {t`Use mods`}
        </Button>
      </Box>
      <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary', mt: -1 }}>
        {t`Queues the mods of a ${{ clientName: form.clientName }} profile into this server profile.`}
      </Typography>
      {queued === null ? null : (
        <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
          {t`${queued} to install.`}
        </Typography>
      )}
    </FormSection>
  )
}
