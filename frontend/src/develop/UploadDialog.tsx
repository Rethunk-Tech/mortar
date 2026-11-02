import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, FormControlLabel, Typography } from '@mui/material'
import { Upload } from 'lucide-react'
import { type ReactNode, useCallback, useEffect, useState } from 'react'
import type { UploadSite } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  SetCurseForgeToken,
  SetThunderstoreToken,
  UploadSites,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { openSettings } from '../nav/store.ts'
import { ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { FormDialog } from '../shell/dialog/FormDialog.tsx'
import { TokenInput } from '../shell/dialog/TokenInput.tsx'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { packProject, runOf, useDevRuns } from './builds.ts'
import { ReleaseEverywhere } from './ReleaseEverywhere.tsx'
import { siteLabel, TEAMS_PAGE, uploadTo } from './uploadSites.ts'

const CURSEFORGE_TOKENS_PAGE = 'https://authors.curseforge.com/#/settings/api-tokens'

// Asks once for a token, which Mortar keeps in the system keyring.
function TokenDialog({
  site,
  subtitle,
  fieldLabel,
  pageLabel,
  page,
  save,
  onSaved,
  onClose,
}: {
  site: string
  subtitle: string
  fieldLabel: string
  pageLabel: string
  page: string
  save: (token: string) => Promise<void>
  onSaved: () => void
  onClose: () => void
}) {
  const { t } = useLingui()
  const [token, setToken] = useState('')
  const [saving, run] = usePending()
  return (
    <FormDialog
      open={true}
      onClose={onClose}
      title={t`Upload to ${{ source: site }}`}
      subtitle={subtitle}
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
        run(
          async () => {
            await save(token.trim())
            onSaved()
          },
          { errorTitle: t`Could not save the token` },
        )
      }
    >
      <TokenInput
        label={fieldLabel}
        value={token}
        onChange={setToken}
        autoFocus={true}
        page={page}
        pageLabel={pageLabel}
      />
    </FormDialog>
  )
}

/** The token dialog for the site that lacks one. */
function AskToken({
  site,
  game,
  dir,
  onSaved,
  onClose,
}: {
  site: string
  game: string
  dir: string
  onSaved: () => void
  onClose: () => void
}) {
  const { t } = useLingui()
  if (site === 'thunderstore') {
    return (
      <TokenDialog
        site={sourceLabel('thunderstore')}
        subtitle={t`Uploads go through a service account of your team.`}
        fieldLabel={t`Service account token`}
        pageLabel={t`Create one in your team's settings`}
        page={TEAMS_PAGE}
        save={(token) => SetThunderstoreToken(game, dir, token)}
        onSaved={onSaved}
        onClose={onClose}
      />
    )
  }
  return (
    <TokenDialog
      site={sourceLabel('curseforge')}
      subtitle={t`One upload token covers every project of yours.`}
      fieldLabel={t`Upload token`}
      pageLabel={t`Create one in your author settings`}
      page={CURSEFORGE_TOKENS_PAGE}
      save={SetCurseForgeToken}
      onSaved={onSaved}
      onClose={onClose}
    />
  )
}

function SiteRow({
  site,
  on,
  onTick,
  onAsk,
  onKey,
}: {
  site: UploadSite
  on: boolean
  onTick: (on: boolean) => void
  onAsk: () => void
  onKey: () => void
}) {
  const { t } = useLingui()
  let missing: ReactNode = null
  if (!site.ready && site.id === 'github') {
    missing = (
      <Typography variant="caption" color="text.secondary">
        {t`Sign in with the gh CLI: gh auth login`}
      </Typography>
    )
  } else if (!site.ready) {
    missing = (
      <Button size="small" onClick={site.id === 'nexus' ? onKey : onAsk}>
        {site.id === 'nexus' ? t`Add your API key in Settings` : t`Add token`}
      </Button>
    )
  }
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, minHeight: 36 }}>
      <FormControlLabel
        sx={{ flex: 1, m: 0 }}
        label={siteLabel(site.id)}
        control={
          <Checkbox
            size="small"
            disabled={!site.ready}
            checked={on}
            onChange={(_, checked) => onTick(checked)}
          />
        }
      />
      {missing}
    </Box>
  )
}

/** After a release that stayed local, or from the menu: lists the sites the project can be uploaded to, asks for the
 * credentials one lacks, then on confirmation builds the package once and uploads it to each site ticked. Nothing is
 * uploaded until the click. */
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
  const [sites, setSites] = useState<UploadSite[] | null>(null)
  const [skipped, setSkipped] = useState<string[]>([])
  const [asking, setAsking] = useState('')
  const [releasing, setReleasing] = useState(false)
  const [uploading, upload] = usePending()
  const load = useCallback(() => {
    UploadSites(game, dir)
      .then((list) => setSites(list ?? []))
      .catch((e: unknown) => {
        setSites([])
        reportError(t`Could not read the sites to upload to`)(e)
      })
  }, [game, dir, t])
  useEffect(load, [load])
  if (sites === null) {
    return null
  }
  const saved = () => {
    setAsking('')
    load()
  }
  if (asking !== '') {
    return (
      <AskToken site={asking} game={game} dir={dir} onSaved={saved} onClose={() => setAsking('')} />
    )
  }
  if (releasing) {
    return (
      <ReleaseEverywhere
        game={game}
        dir={dir}
        name={name}
        version={version}
        sites={sites}
        onClose={onClose}
      />
    )
  }
  const offered = sites.filter((s) => s.configured)
  const chosen = offered.filter((s) => s.ready && !skipped.includes(s.id))
  return (
    <FormDialog
      open={true}
      onClose={onClose}
      title={t`Upload ${{ name }} ${{ version }}`}
      subtitle={t`Mortar builds the package once and uploads it to each site ticked. A published version cannot be replaced.`}
      icon={
        <ToneIcon tone="neutral">
          <Upload size={22} />
        </ToneIcon>
      }
      size="sm"
      submitLabel={uploading ? t`Uploading…` : t`Upload`}
      issue={chosen.length === 0 ? t`Tick a site that has its credentials.` : ''}
      dirty={false}
      busy={uploading}
      onSubmit={() =>
        upload(
          async () => {
            if (runOf(useDevRuns.getState().byDir, dir).running) {
              useToasts
                .getState()
                .push({ kind: 'info', title: t`Wait for the running build to finish` })
              return
            }
            const pack = await packProject(game, dir)
            // The card shows why a package was refused.
            if (!pack?.zip || (pack.issues ?? []).length > 0) {
              onClose()
              return
            }
            for (const s of chosen) {
              await uploadTo(s.id, { game, dir, name, version })
            }
            onClose()
          },
          { errorTitle: t`Could not upload the mod` },
        )
      }
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
        {offered.map((s) => (
          <SiteRow
            key={s.id}
            site={s}
            on={s.ready && !skipped.includes(s.id)}
            onTick={(on) =>
              setSkipped((list) => (on ? list.filter((x) => x !== s.id) : [...list, s.id]))
            }
            onAsk={() => setAsking(s.id)}
            onKey={() => {
              onClose()
              openSettings('accounts')
            }}
          />
        ))}
        <Box>
          <Button
            size="small"
            disabled={uploading || !offered.some((s) => s.ready)}
            onClick={() => setReleasing(true)}
          >
            {t`Release everywhere`}
          </Button>
        </Box>
      </Box>
    </FormDialog>
  )
}
