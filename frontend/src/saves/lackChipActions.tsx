import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { ExternalLink, Plus, Power, X } from 'lucide-react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { CopyMods } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import type {
  Fit,
  Lack,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/models.ts'
import { LockedReason } from '../mods/LockedReason.tsx'
import { openPage } from '../mods/menu.ts'
import { download } from '../queue/actions.ts'
import { TipIconButton } from '../shell/TipIconButton.tsx'
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
        <TipIconButton
          label={locked ? t`Stop the game to change mods.` : t`Switch on ${name} in this profile`}
          disabled={locked}
          onClick={() => {
            enable(game, profile, lack.id).catch(reportUnexpected)
          }}
        >
          <Power size={14} />
        </TipIconButton>
      ) : null}
      {want ? (
        <TipIconButton
          label={plusTitle}
          disabled={queued || locked}
          onClick={() => {
            download([want]).catch(reportUnexpected)
          }}
        >
          <Plus size={14} />
        </TipIconButton>
      ) : null}
      {source ? (
        <LockedReason locked={locked}>
          <Button
            size="small"
            disabled={copying || locked}
            title={source.name}
            aria-label={t`Copy ${source.name}`}
            onClick={() => runCopy(() => CopyMods(game, source.id, profile.id, [lack.id]))}
            sx={{ minWidth: 0, px: 0.5, ...nowrap }}
          >
            {t`Copy`}
          </Button>
        </LockedReason>
      ) : null}
      {!(lack.disabled || want) && url ? (
        <TipIconButton
          label={nexusPage ? t`Open ${name} on Nexus` : t`Open the page of ${name}`}
          onClick={() => {
            openPage(url).catch(reportUnexpected)
          }}
        >
          <ExternalLink size={14} />
        </TipIconButton>
      ) : null}
      <TipIconButton
        label={t`Dismiss ${name} for this save`}
        onClick={() => {
          dismiss(fit.folder, lack.id).catch(reportUnexpected)
        }}
      >
        <X size={14} />
      </TipIconButton>
    </>
  )
}
