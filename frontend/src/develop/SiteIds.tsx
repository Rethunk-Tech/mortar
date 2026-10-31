import { useLingui } from '@lingui/react/macro'
import type {
  CurseForgeIDs,
  NexusIDs,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { FieldRow, FormSection, TextInput } from '../shell/dialog/FormDialog.tsx'

// Nine digits is past every CurseForge project id and well inside what the Go side accepts.
const MAX_DIGITS = 9

const MAX_NEXUS_ID = 64
const nexusId = (text: string) => text.replace(/[^A-Za-z0-9_-]/g, '').slice(0, MAX_NEXUS_ID)
const projectText = (id: number | undefined) => (id ? String(id) : '')
const projectId = (text: string) => Number(text.replace(/\D/g, '').slice(0, MAX_DIGITS)) || 0

/** The Edit dialog's ids for the sites Release uploads to besides Thunderstore; a site with no ids is not offered. */
export function SiteIds({
  sites,
  nexus,
  curseforge,
  onNexus,
  onCurseForge,
}: {
  // The ids of the sites the game can be uploaded to.
  sites: string[]
  nexus: NexusIDs
  curseforge: CurseForgeIDs
  onNexus: (ids: NexusIDs) => void
  onCurseForge: (ids: CurseForgeIDs) => void
}) {
  const { t } = useLingui()
  const hasNexus = sites.includes('nexus')
  const hasCurseForge = sites.includes('curseforge')
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
            value={nexus.modId ?? ''}
            hint={t`Adds the changelog and links the page.`}
            onChange={(v) => onNexus({ ...nexus, modId: nexusId(v) })}
          />
          <TextInput
            label={t`${{ source: sourceLabel('nexus') }} file id`}
            value={nexus.fileId ?? ''}
            hint={t`Shown under Advanced on the mod's Files tab.`}
            onChange={(v) => onNexus({ ...nexus, fileId: nexusId(v) })}
          />
        </FieldRow>
      ) : null}
      {hasCurseForge ? (
        <TextInput
          label={t`${{ source: sourceLabel('curseforge') }} project id`}
          value={projectText(curseforge.projectId)}
          hint={t`Shown on the project's overview page.`}
          onChange={(v) => onCurseForge({ projectId: projectId(v) })}
        />
      ) : null}
    </FormSection>
  )
}
