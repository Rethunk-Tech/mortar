import { useLingui } from '@lingui/react/macro'
import { Button, IconButton, Tooltip } from '@mui/material'
import { Browser } from '@wailsio/runtime'
import { ExternalLink, Plus, Power, X } from 'lucide-react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { CopyMods } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import type {
  Fit,
  Lack,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/savessvc/models.ts'
import { download } from '../queue/actions.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { useSaves } from './store.ts'
import { wantFor } from './wantFor.ts'

const nowrap = { whiteSpace: 'nowrap' } as const

export function LackChipActions(p: {
  fit: Fit
  lack: Lack
  profile: Profile
  game: string
  locked: boolean
  queued: boolean
  plusTitle: string
  source: { id: string; name: string } | null
}) {
  const { fit, lack, profile, game, locked, queued, plusTitle, source } = p
  const { t } = useLingui()
  const { name } = lack
  const dismiss = useSaves((s) => s.dismiss)
  const enable = useSaves((s) => s.enable)
  const [copying, runCopy] = usePending()
  const want = lack.disabled ? null : wantFor(lack)
  const url = lack.where?.url ?? ''
  const nexusPage = lack.where?.site === 'Nexus'
  return (
    <>
      {lack.disabled ? (
        <Tooltip title={locked ? t`Stop the game to change mods.` : t`Switch on in this profile`}>
          <span>
            <IconButton
              size="small"
              disabled={locked}
              aria-label={t`Switch on ${name} in this profile`}
              onClick={() => {
                enable(game, profile, lack.uniqueId).catch(reportUnexpected)
              }}
            >
              <Power size={14} />
            </IconButton>
          </span>
        </Tooltip>
      ) : null}
      {want ? (
        <Tooltip title={plusTitle}>
          <span>
            <IconButton
              size="small"
              disabled={queued || locked}
              aria-label={t`Add ${name} to this profile`}
              onClick={() => {
                download([want]).catch(reportUnexpected)
              }}
            >
              <Plus size={14} />
            </IconButton>
          </span>
        </Tooltip>
      ) : null}
      {source ? (
        <DisabledReason title={t`Stop the game to change mods.`} disabled={locked}>
          <Button
            size="small"
            disabled={copying || locked}
            title={source.name}
            aria-label={t`Copy ${source.name}`}
            onClick={() => runCopy(() => CopyMods(game, source.id, profile.id, [lack.uniqueId]))}
            sx={{ minWidth: 0, px: 0.5, ...nowrap }}
          >
            {t`Copy`}
          </Button>
        </DisabledReason>
      ) : null}
      {!(lack.disabled || want) && url ? (
        <Tooltip title={nexusPage ? t`Open on Nexus` : t`Open page`}>
          <IconButton
            size="small"
            aria-label={nexusPage ? t`Open ${name} on Nexus` : t`Open the page of ${name}`}
            onClick={() => {
              Browser.OpenURL(url).catch(reportUnexpected)
            }}
          >
            <ExternalLink size={14} />
          </IconButton>
        </Tooltip>
      ) : null}
      <Tooltip title={t`Dismiss for this save`}>
        <IconButton
          size="small"
          aria-label={t`Dismiss ${name} for this save`}
          onClick={() => {
            dismiss(fit.folder, lack.uniqueId).catch(reportUnexpected)
          }}
        >
          <X size={14} />
        </IconButton>
      </Tooltip>
    </>
  )
}
