import { useLingui } from '@lingui/react/macro'
import { LayoutTemplate } from 'lucide-react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { useTemplates } from '../templates/useTemplates.ts'
import { openProfileDialog } from './profileDialog.ts'

interface Props {
  profile: Pick<Profile, 'id'>
  close: () => void
}

function SaveTemplateMenuItem({ profile, close }: Props) {
  const { t } = useLingui()
  const hasGame = useProfiles((s) => s.game !== null)
  return (
    <MenuAction
      icon={<LayoutTemplate size={16} />}
      label={t`Save as template…`}
      disabled={!hasGame}
      onClick={() => {
        close()
        openProfileDialog('saveTemplate', profile.id)
      }}
    />
  )
}

function ApplyTemplateMenuItem({ profile, close }: Props) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const { templates } = useTemplates(game, true)
  return templates.length === 0 ? null : (
    <MenuAction
      icon={<LayoutTemplate size={16} />}
      label={t`Apply a template…`}
      onClick={() => {
        close()
        openProfileDialog('applyTemplate', profile.id)
      }}
    />
  )
}

function ManageTemplatesMenuItem({ profile, close }: Props) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const { templates } = useTemplates(game, true)
  return templates.length === 0 ? null : (
    <MenuAction
      icon={<LayoutTemplate size={16} />}
      label={t`Manage templates…`}
      onClick={() => {
        close()
        openProfileDialog('manageTemplates', profile.id)
      }}
    />
  )
}

export { ApplyTemplateMenuItem, ManageTemplatesMenuItem, SaveTemplateMenuItem }
