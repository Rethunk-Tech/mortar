import { SaveTemplateDialog } from '../templates/TemplateDialogs.tsx'
import { useSaveImported } from './savePrompt.ts'

export function SaveImportedTemplateHost() {
  const target = useSaveImported((s) => s.target)
  return (
    <SaveTemplateDialog
      open={target !== null}
      game={target?.game ?? ''}
      profileId={target?.profileId ?? ''}
      profileName={target?.name ?? ''}
      onClose={() => useSaveImported.setState({ target: null })}
    />
  )
}
