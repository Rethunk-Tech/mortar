import { useLingui } from '@lingui/react/macro'
import type {
  CurseForgeIDs,
  NexusIDs,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { FieldRow, FormSection, TextInput } from '../shell/dialog/FormDialog.tsx'

// Nine digits is past every id the sites hand out and well inside what the Go side accepts.
const MAX_DIGITS = 9

const idText = (id: number | undefined) => (id ? String(id) : '')
const idOf = (text: string) => Number(text.replace(/\D/g, '').slice(0, MAX_DIGITS)) || 0

/** The Edit dialog's ids for the sites Release uploads to besides Thunderstore; a site with no ids is not offered. */
export function SiteIds({
  sources,
  nexus,
  curseforge,
  onNexus,
  onCurseForge,
}: {
  sources: string[]
  nexus: NexusIDs
  curseforge: CurseForgeIDs
  onNexus: (ids: NexusIDs) => void
  onCurseForge: (ids: CurseForgeIDs) => void
}) {
  const { t } = useLingui()
  const hasNexus = sources.includes('nexus')
  const hasCurseForge = sources.includes('curseforge')
  if (!(hasNexus || hasCurseForge)) {
    return null
  }
  return (
    <FormSection
      title={t`Other sites`}
      description={t`Release uploads to a site once its ids are set. Leave them empty to skip it.`}
    >
      {hasNexus ? (
        <FieldRow>
          <TextInput
            label={t`${{ source: sourceLabel('nexus') }} mod id`}
            value={idText(nexus.modId)}
            hint={t`Adds the changelog and links the page.`}
            onChange={(v) => onNexus({ ...nexus, modId: idOf(v) })}
          />
          <TextInput
            label={t`${{ source: sourceLabel('nexus') }} file id`}
            value={idText(nexus.fileId)}
            hint={t`Shown under Advanced on the mod's Files tab.`}
            onChange={(v) => onNexus({ ...nexus, fileId: idOf(v) })}
          />
        </FieldRow>
      ) : null}
      {hasCurseForge ? (
        <TextInput
          label={t`${{ source: sourceLabel('curseforge') }} project id`}
          value={idText(curseforge.projectId)}
          hint={t`Shown on the project's overview page.`}
          onChange={(v) => onCurseForge({ projectId: idOf(v) })}
        />
      ) : null}
    </FormSection>
  )
}
