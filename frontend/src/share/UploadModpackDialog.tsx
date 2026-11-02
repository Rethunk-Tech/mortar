import { useLingui } from '@lingui/react/macro'
import { Button, Typography } from '@mui/material'
import { useLoaded } from '@rethunk/react-load-hooks'
import { Upload } from 'lucide-react'
import { useState } from 'react'
import type { ModpackOptions } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/models.ts'
import {
  ForgetThunderstoreToken,
  HasThunderstoreToken,
  SetThunderstoreToken,
  ThunderstoreCategories,
  UploadModpack,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { ChoicePicker } from '../develop/ChoicePicker.tsx'
import { TEAMS_PAGE } from '../develop/uploadSites.ts'
import { ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { FormDialog, TextInput } from '../shell/dialog/FormDialog.tsx'
import { openWeb } from '../shell/openWeb.ts'
import { errorKind } from '../toasts/errorKind.ts'
import { errorMessage, reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'

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
  const [team, setTeam] = useState(options.team || options.author)
  const [categories, setCategories] = useState<string[]>(['modpacks'])
  const [token, setToken] = useState('')
  const [busy, run] = usePending()
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
        try {
          const url = await UploadModpack(game, profileId, options, name, categories)
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
      <CategoryField game={game} value={categories} onChange={setCategories} />
      <TextInput
        label={t`Service account token`}
        type="password"
        value={token}
        onChange={setToken}
        placeholder={hasToken ? t`Kept in the system keyring` : ''}
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
      />
    </FormDialog>
  )
}

/** The community's categories as a multi-select, or a comma separated field with the failure shown when the list cannot be loaded. */
function CategoryField({
  game,
  value,
  onChange,
}: {
  game: string
  value: string[]
  onChange: (next: string[]) => void
}) {
  const { t } = useLingui()
  const listed = useLoaded<{ value: string; label: string }[]>(
    () =>
      ThunderstoreCategories(game).then((l) =>
        (l ?? []).map((c) => ({ value: c.slug, label: c.name })),
      ),
    [game],
    [],
  )
  if (!listed.error) {
    return (
      <ChoicePicker label={t`Categories`} choices={listed.data} value={value} onChange={onChange} />
    )
  }
  return <TypedCategories value={value} onChange={onChange} failure={errorMessage(listed.error)} />
}

function TypedCategories({
  value,
  onChange,
  failure,
}: {
  value: string[]
  onChange: (next: string[]) => void
  failure: string
}) {
  const { t } = useLingui()
  const [text, setText] = useState(value.join(', '))
  return (
    <>
      <TextInput
        label={t`Categories`}
        value={text}
        onChange={(next) => {
          setText(next)
          onChange(
            next
              .split(',')
              .map((c) => c.trim())
              .filter(Boolean),
          )
        }}
        hint={t`Separated by commas.`}
      />
      <Typography sx={{ fontSize: 'base.fontSize', color: 'text.secondary' }}>
        {t`Could not load the category list. ${failure}`}
      </Typography>
    </>
  )
}
