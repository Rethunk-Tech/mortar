import { useLingui } from '@lingui/react/macro'
import { LayoutTemplate } from 'lucide-react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { useTemplates } from '../templates/useTemplates.ts'
import { openProfileDialog, type ProfileDialogKind } from './profileDialog.ts'

interface Props {
  profile: Pick<Profile, 'id'>
  close: () => void
}

function TemplateAction({
  profile,
  close,
  kind,
  label,
  disabled = false,
}: Props & { kind: ProfileDialogKind; label: string; disabled?: boolean }) {
  return (
    <MenuAction
      icon={<LayoutTemplate size={16} />}
      label={label}
      disabled={disabled}
      onClick={() => {
        close()
        openProfileDialog(kind, profile.id)
      }}
    />
  )
}

function useHasTemplates() {
  const game = useProfiles((s) => s.game?.id ?? '')
  return useTemplates(game, true).templates.length > 0
}

function SaveTemplateMenuItem(props: Props) {
  const { t } = useLingui()
  const hasGame = useProfiles((s) => s.game !== null)
  return (
    <TemplateAction
      {...props}
      kind="saveTemplate"
      label={t`Save as template…`}
      disabled={!hasGame}
    />
  )
}

function ApplyTemplateMenuItem(props: Props) {
  const { t } = useLingui()
  return useHasTemplates() ? (
    <TemplateAction {...props} kind="applyTemplate" label={t`Apply a template…`} />
  ) : null
}

function ManageTemplatesMenuItem(props: Props) {
  const { t } = useLingui()
  return useHasTemplates() ? (
    <TemplateAction {...props} kind="manageTemplates" label={t`Manage templates…`} />
  ) : null
}

export { ApplyTemplateMenuItem, ManageTemplatesMenuItem, SaveTemplateMenuItem }
