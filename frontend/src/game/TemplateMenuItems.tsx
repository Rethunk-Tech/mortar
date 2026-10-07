import { useLingui } from '@lingui/react/macro'
import { LayoutTemplate } from 'lucide-react'
import { useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { ApplyTemplateDialog } from '../templates/ApplyTemplateDialog.tsx'
import { ManageTemplatesDialog, SaveTemplateDialog } from '../templates/TemplateDialogs.tsx'
import { useTemplates } from '../templates/useTemplates.ts'

interface Props {
  profile: Profile
  close: () => void
}

function SaveTemplateMenuItem({ profile, close }: Props) {
  const { t } = useLingui()
  const currentGame = useProfiles((s) => s.game)
  const [open, setOpen] = useState(false)
  return (
    <>
      <MenuAction
        icon={<LayoutTemplate size={16} />}
        label={t`Save as template…`}
        disabled={!currentGame}
        onClick={() => {
          close()
          setOpen(true)
        }}
      />
      <SaveTemplateDialog
        open={open}
        game={currentGame?.id ?? ''}
        profileId={profile.id}
        profileName={profile.name}
        onClose={() => setOpen(false)}
      />
    </>
  )
}

function ApplyTemplateMenuItem({ profile, close }: Props) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const { templates } = useTemplates(game, true)
  const [open, setOpen] = useState(false)
  return templates.length === 0 ? null : (
    <>
      <MenuAction
        icon={<LayoutTemplate size={16} />}
        label={t`Apply a template…`}
        onClick={() => {
          close()
          setOpen(true)
        }}
      />
      <ApplyTemplateDialog
        open={open}
        game={game}
        profileId={profile.id}
        templates={templates}
        onClose={() => setOpen(false)}
      />
    </>
  )
}

function ManageTemplatesMenuItem({ close }: Pick<Props, 'close'>) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const { templates, reload } = useTemplates(game, true)
  const [open, setOpen] = useState(false)
  return templates.length === 0 ? null : (
    <>
      <MenuAction
        icon={<LayoutTemplate size={16} />}
        label={t`Manage templates…`}
        onClick={() => {
          close()
          setOpen(true)
        }}
      />
      <ManageTemplatesDialog
        open={open}
        game={game}
        templates={templates}
        onChanged={reload}
        onClose={() => setOpen(false)}
      />
    </>
  )
}

export function TemplateMenuItems({ profile, close }: Props) {
  return [
    <ApplyTemplateMenuItem key="apply-template" profile={profile} close={close} />,
    <SaveTemplateMenuItem key="save-template" profile={profile} close={close} />,
    <ManageTemplatesMenuItem key="manage-templates" close={close} />,
  ]
}

export { SaveTemplateMenuItem }
