import { useLingui } from '@lingui/react/macro'
import { Typography } from '@mui/material'
import { useCallback, useRef } from 'react'
import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/source/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { FormSection } from '../shell/dialog/FormDialog.tsx'
import { useProjectInfo } from './builds.ts'
import { AddAnother, PackageRow } from './DependenciesStep.tsx'
import { knowPackage, ownProject, useKnownPackages, usePackageDetails } from './dependencies.ts'
import type { DependencyRow } from './projectDependencies.ts'

/** The Edit dialog's dependency list: untick to drop one, switch it between required and optional, or search to add. */
export function ProjectDependencies({
  game,
  rows,
  setRows,
}: {
  game: string
  rows: DependencyRow[]
  setRows: (update: (rows: DependencyRow[]) => DependencyRow[]) => void
}) {
  const { t } = useLingui()
  const profile = useProfiles((s) => s.openId)
  const known = useKnownPackages((s) => s)
  const projects = useProjectInfo((s) => s.byDir)
  const ids = rows.map((r) => r.package)
  usePackageDetails(game, profile, ids)
  const latest = useRef(setRows)
  latest.current = setRows
  const toggle = useCallback((id: string, on: boolean) => {
    latest.current((now) =>
      on || now.find((r) => r.package === id)?.added === false
        ? now.map((r) => (r.package === id ? { ...r, keep: on } : r))
        : now.filter((r) => r.package !== id),
    )
  }, [])
  const setOptional = useCallback((id: string, soft: boolean) => {
    latest.current((now) => now.map((r) => (r.package === id ? { ...r, soft } : r)))
  }, [])
  const add = useCallback((item: Item) => {
    knowPackage(item)
    latest.current((now) => [
      ...now,
      { package: item.id, version: item.version, soft: false, keep: true, added: true },
    ])
  }, [])
  return (
    <FormSection
      title={t`Dependencies`}
      description={t`Untick one to drop it; new ones install into the mod's profile.`}
    >
      {rows.length === 0 ? (
        <Typography
          sx={{ color: 'text.secondary' }}
        >{t`The mod depends on nothing but the loader.`}</Typography>
      ) : null}
      {rows.map((r) => (
        <PackageRow
          key={r.package}
          id={r.package}
          item={known[r.package] ?? ownProject(projects, r.package)}
          why={r.added ? t`New` : ''}
          checked={r.keep}
          onToggle={toggle}
          optional={r.soft}
          onOptional={setOptional}
        />
      ))}
      <AddAnother game={game} profile={profile} taken={ids.join(',')} onAdd={add} />
    </FormSection>
  )
}
