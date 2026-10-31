import { useLingui } from '@lingui/react/macro'
import { Typography } from '@mui/material'
import { useCallback, useRef } from 'react'
import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/source/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { FormSection } from '../shell/dialog/FormDialog.tsx'
import { useProjectInfo } from './builds.ts'
import { AddAnother, PackageRow } from './DependenciesStep.tsx'
import { knowPackage, ownProject, useKnownPackages, usePackageDetails } from './dependencies.ts'

/** The Edit dialog's list of mods Test installs beside the project: untick to drop one, search to add. They stay out
 * of the package. */
export function TestWith({
  game,
  ids,
  onChange,
}: {
  game: string
  ids: string[]
  onChange: (ids: string[]) => void
}) {
  const { t } = useLingui()
  const profile = useProfiles((s) => s.openId)
  const known = useKnownPackages((s) => s)
  const projects = useProjectInfo((s) => s.byDir)
  usePackageDetails(game, profile, ids)
  const latest = useRef({ ids, onChange })
  latest.current = { ids, onChange }
  const toggle = useCallback((id: string, on: boolean) => {
    const { ids: now, onChange: apply } = latest.current
    apply(on ? now : now.filter((x) => x !== id))
  }, [])
  const add = useCallback((item: Item) => {
    knowPackage(item)
    const { ids: now, onChange: apply } = latest.current
    apply([...now, item.id])
  }, [])
  return (
    <FormSection
      title={t`Test with`}
      description={t`Mods that Test installs into the profile beside this one. They are never part of the package.`}
    >
      {ids.length === 0 ? (
        <Typography sx={{ color: 'text.secondary' }}>{t`Test installs nothing else.`}</Typography>
      ) : null}
      {ids.map((id) => (
        <PackageRow
          key={id}
          id={id}
          item={known[id] ?? ownProject(projects, id)}
          why=""
          checked={true}
          onToggle={toggle}
          optional={false}
        />
      ))}
      <AddAnother game={game} profile={profile} taken={ids.join(',')} onAdd={add} />
    </FormSection>
  )
}
