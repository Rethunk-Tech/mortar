import { useLingui } from '@lingui/react/macro'
import { Button, TextField } from '@mui/material'
import { Upload } from 'lucide-react'
import { useEffect, useId, useState } from 'react'
import {
  HasThunderstoreToken,
  SetThunderstoreToken,
  UploadThunderstore,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { Field, FormDialog } from '../shell/dialog/FormDialog.tsx'
import { openWeb } from '../shell/openWeb.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { packProject } from './builds.ts'

const TEAMS_PAGE = 'https://thunderstore.io/settings/teams/'

// Asks once for a team's service account token, which Mortar keeps in the system keyring.
function TokenDialog({
  game,
  dir,
  onSaved,
  onClose,
}: {
  game: string
  dir: string
  onSaved: () => void
  onClose: () => void
}) {
  const { t } = useLingui()
  const [token, setToken] = useState('')
  const [saving, save] = usePending()
  const id = useId()
  const site = sourceLabel('thunderstore')
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
      submitLabel={t`Save token`}
      issue={token.trim() === '' ? t`Paste the token.` : ''}
      dirty={token !== ''}
      busy={saving}
      onSubmit={() =>
        save(
          async () => {
            await SetThunderstoreToken(game, dir, token.trim())
            onSaved()
          },
          { errorTitle: t`Could not save the token` },
        )
      }
    >
      <Field
        label={t`Service account token`}
        htmlFor={id}
        hint={
          <Button
            size="small"
            sx={{ px: 0 }}
            onClick={() => openWeb(TEAMS_PAGE).catch(reportError(t`Could not open the page`))}
          >
            {t`Create one in your team's settings`}
          </Button>
        }
      >
        <TextField
          id={id}
          type="password"
          size="small"
          fullWidth={true}
          autoFocus={true}
          value={token}
          onChange={(e) => setToken(e.target.value)}
        />
      </Field>
    </FormDialog>
  )
}

/** After a release that stayed local: asks for the token once, then on confirmation builds the package and publishes
 * it to the game's Thunderstore community. */
export function UploadDialog({
  game,
  dir,
  name,
  version,
  onClose,
}: {
  game: string
  dir: string
  name: string
  version: string
  onClose: () => void
}) {
  const { t } = useLingui()
  const [hasToken, setHasToken] = useState<boolean | null>(null)
  const [uploading, upload] = usePending()
  const site = sourceLabel('thunderstore')
  useEffect(() => {
    HasThunderstoreToken(game, dir)
      .then(setHasToken)
      .catch(() => setHasToken(false))
  }, [game, dir])
  if (hasToken === null) {
    return null
  }
  if (!hasToken) {
    return <TokenDialog game={game} dir={dir} onSaved={() => setHasToken(true)} onClose={onClose} />
  }
  return (
    <ConfirmationDialog
      open={true}
      title={t`Upload ${{ name }} ${{ version }} to ${{ source: site }}?`}
      body={t`Mortar builds the package and publishes it for everyone under your team. A published version cannot be replaced.`}
      confirmLabel={uploading ? t`Uploading…` : t`Upload`}
      busy={uploading}
      onCancel={onClose}
      onConfirm={() =>
        upload(
          async () => {
            const pack = await packProject(game, dir)
            // The card shows why a package was refused.
            if (!pack?.zip || (pack.issues ?? []).length > 0) {
              onClose()
              return
            }
            const url = await UploadThunderstore(game, dir)
            useToasts.getState().push({
              kind: 'success',
              title: t`Uploaded ${{ name }}`,
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
          },
          { errorTitle: t`Could not upload the mod` },
        )
      }
    />
  )
}
