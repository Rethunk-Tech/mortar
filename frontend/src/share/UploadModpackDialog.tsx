import { useLingui } from '@lingui/react/macro'
import { Button, TextField } from '@mui/material'
import { useLoaded } from '@rethunk/react-load-hooks'
import { Upload } from 'lucide-react'
import { useId, useState } from 'react'
import type { ModpackOptions } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/models.ts'
import {
  ForgetThunderstoreToken,
  HasThunderstoreToken,
  SetThunderstoreToken,
  UploadModpack,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { Field, FormDialog, TextInput } from '../shell/dialog/FormDialog.tsx'
import { openWeb } from '../shell/openWeb.ts'
import { errorKind } from '../toasts/errorKind.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'

const TEAMS_PAGE = 'https://thunderstore.io/settings/teams/'

/** Uploads the modpack to Thunderstore under a team, with that team's service account token kept in the keyring. */
export function UploadModpackDialog({
  game,
  profileId,
  options,
  onClose,
}: {
  game: string
  profileId: string
  options: ModpackOptions
  onClose: () => void
}) {
  const { t } = useLingui()
  const [team, setTeam] = useState(options.author)
  const [categories, setCategories] = useState('modpacks')
  const [token, setToken] = useState('')
  const [busy, run] = usePending()
  const id = useId()
  const site = sourceLabel('thunderstore')
  const kept = useLoaded<boolean>(
    () => (team.trim() === '' ? Promise.resolve(false) : HasThunderstoreToken(team.trim())),
    [team],
    false,
  )
  const hasToken = kept.data && token === ''
  let issue = ''
  if (team.trim() === '') {
    issue = t`Name the team to upload under.`
  } else if (!kept.data && token.trim() === '') {
    issue = t`Paste the team's service account token.`
  }
  const submit = () =>
    run(
      async () => {
        const name = team.trim()
        if (token.trim() !== '') {
          await SetThunderstoreToken(name, token.trim())
        }
        const list = categories
          .split(',')
          .map((c) => c.trim())
          .filter(Boolean)
        try {
          const url = await UploadModpack(game, profileId, options, name, list)
          useToasts.getState().push({
            kind: 'success',
            title: t`Uploaded ${{ name: options.name }} to ${{ site }}`,
            ...(url
              ? {
                  action: {
                    label: t`Open page`,
                    run: () => openWeb(url).catch(reportError(t`Could not open the page`)),
                  },
                }
              : {}),
          })
          onClose()
        } catch (e) {
          // A refused token is forgotten, so the dialog asks for it again.
          if (errorKind(e) === 'permission') {
            await ForgetThunderstoreToken(name).catch(() => undefined)
            kept.reload()
          }
          throw e
        }
      },
      { errorTitle: t`Could not upload to ${{ site }}` },
    )
  return (
    <FormDialog
      open={true}
      onClose={onClose}
      title={t`Upload to ${{ source: site }}`}
      subtitle={t`Uploads go through a service account of your team.`}
      icon={
        <ToneIcon tone="neutral">
          <Upload size={22} />
        </ToneIcon>
      }
      size="sm"
      submitLabel={t`Upload`}
      issue={issue}
      dirty={false}
      busy={busy}
      onSubmit={submit}
    >
      <TextInput label={t`Team`} value={team} onChange={setTeam} autoFocus={true} />
      <TextInput
        label={t`Categories`}
        value={categories}
        onChange={setCategories}
        hint={t`Separated by commas.`}
      />
      <Field
        label={t`Service account token`}
        htmlFor={id}
        hint={
          hasToken ? (
            <Button
              size="small"
              sx={{ px: 0 }}
              onClick={() =>
                ForgetThunderstoreToken(team.trim())
                  .then(kept.reload)
                  .catch(reportError(t`Could not forget the token`))
              }
            >
              {t`A token is kept. Forget it`}
            </Button>
          ) : (
            <Button
              size="small"
              sx={{ px: 0 }}
              onClick={() => openWeb(TEAMS_PAGE).catch(reportError(t`Could not open the page`))}
            >
              {t`Create one in your team's settings`}
            </Button>
          )
        }
      >
        <TextField
          id={id}
          type="password"
          size="small"
          fullWidth={true}
          value={token}
          placeholder={hasToken ? t`Kept in the system keyring` : ''}
          onChange={(e) => setToken(e.target.value)}
        />
      </Field>
    </FormDialog>
  )
}
