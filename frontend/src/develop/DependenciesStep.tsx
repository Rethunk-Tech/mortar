import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, MenuItem, TextField, Typography } from '@mui/material'
import { memo, useCallback, useRef, useState } from 'react'
import { Search } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'
import type { Form } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/source/models.ts'
import { lowerFirst } from '../i18n/lowerFirst.ts'
import { LetterTile } from '../mods/LetterTile.tsx'
import { FormSection } from '../shell/dialog/FormDialog.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { useDebouncedLoad } from '../shell/useDebouncedLoad.ts'
import { space } from '../theme/density.ts'
import { useProjectInfo } from './builds.ts'
import {
  knowPackage,
  NO_FILTER,
  ownProject,
  type Shown,
  THUNDERSTORE,
  useKnownPackages,
  usePackageDetails,
} from './dependencies.ts'
import { chosenDependencies, type Draft, type Recommended, recommendations } from './draft.ts'

const SEARCH_HITS = 5
const SEARCH_DELAY_MS = 250

// The answer that brought a package in, as the player saw it: the choice's label, or the switch's.
function answerLabel(form: Form | null, r: Recommended): string {
  const param = form?.parameters?.find((p) => p.id === r.param)
  if (!param) {
    return r.value
  }
  return param.choices?.find((c) => c.value === r.value)?.label ?? param.label
}

// Memoised: a package's details arriving, or one row's tick, redraws only that row.
export const PackageRow = memo(function PackageRowView({
  id,
  item,
  why,
  checked,
  onToggle,
  optional,
  onOptional,
}: {
  id: string
  item: Shown | undefined
  why: string
  checked: boolean
  onToggle: (id: string, on: boolean) => void
  optional: boolean
  // Left out for a mod of files, whose dependencies are all required.
  onOptional?: (id: string, on: boolean) => void
}) {
  const { t } = useLingui()
  const name = item?.name ?? id.slice(id.indexOf('-') + 1).replaceAll('_', ' ')
  return (
    <Box
      component="label"
      sx={{ display: 'flex', alignItems: 'center', gap: space.gap, cursor: 'pointer' }}
    >
      <Checkbox checked={checked} onChange={(_, on) => onToggle(id, on)} />
      <LetterTile
        mod={{ id, name, ...(item?.picture ? { picture: item.picture } : {}) }}
        size={32}
      />
      <Box sx={{ minWidth: 0, flex: 1 }}>
        <Typography noWrap={true} sx={{ fontWeight: 600 }}>
          {name}
        </Typography>
        <Typography noWrap={true} sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
          {[item?.author, why].filter(Boolean).join(' · ')}
        </Typography>
      </Box>
      {onOptional ? (
        <TextField
          select={true}
          size="small"
          disabled={!checked}
          value={optional ? 'soft' : 'hard'}
          onChange={(e) => onOptional(id, e.target.value === 'soft')}
          slotProps={{ htmlInput: { 'aria-label': t`How ${{ name }} is needed` } }}
          sx={{ width: 140 }}
        >
          <MenuItem value="hard">{t`Required`}</MenuItem>
          <MenuItem value="soft">{t`Optional`}</MenuItem>
        </TextField>
      ) : null}
    </Box>
  )
})

// One empty list, so clearing the hits when they are already clear sets nothing.
const NO_HITS: Item[] = []

function useSearch(game: string, profile: string, query: string) {
  const searching = query.trim().length >= 2
  // A failed search shows no hits and stays quiet: the field is a suggestion list, not an action.
  return useDebouncedLoad<Item[]>(
    searching
      ? () =>
          Search(game, THUNDERSTORE, query, 1, profile, NO_FILTER).then(
            (page) => (page.items ?? []).slice(0, SEARCH_HITS),
            () => NO_HITS,
          )
      : null,
    [game, profile, query, searching],
    { initial: NO_HITS, delayMs: SEARCH_DELAY_MS },
  ).data
}

// The search for more packages, memoised so a package's details arriving above leave it alone. `taken` lists the ids
// already shown, comma-joined so an unchanged list compares equal.
export const AddAnother = memo(function AddAnotherView({
  game,
  profile,
  taken,
  onAdd,
}: {
  game: string
  profile: string
  taken: string
  onAdd: (item: Item) => void
}) {
  const { t } = useLingui()
  const [query, setQuery] = useState('')
  const shown = taken.split(',')
  const hits = useSearch(game, profile, query).filter((h) => !shown.includes(h.id))
  return (
    <>
      <SearchField
        value={query}
        onChange={setQuery}
        label={t`Add another mod`}
        placeholder={t`Add another mod`}
      />
      {hits.map((h) => (
        <Box key={h.id} sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
          <LetterTile
            mod={{ id: h.id, name: h.name, ...(h.picture ? { picture: h.picture } : {}) }}
            size={32}
          />
          <Box sx={{ minWidth: 0, flex: 1 }}>
            <Typography noWrap={true} sx={{ fontWeight: 600 }}>
              {h.name}
            </Typography>
            <Typography noWrap={true} sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
              {h.author}
            </Typography>
          </Box>
          <Button size="small" variant="outlined" onClick={() => onAdd(h)}>
            {t`Add`}
          </Button>
        </Box>
      ))}
    </>
  )
})

/** The packages the mod starts depending on: those the answers suggest, ticked, and any the player searches for. */
export function DependenciesStep({
  game,
  profile,
  form,
  draft,
  set,
}: {
  game: string
  profile: string
  form: Form | null
  draft: Draft
  set: (next: Partial<Draft>) => void
}) {
  const { t } = useLingui()
  const known = useKnownPackages((s) => s)
  const projects = useProjectInfo((s) => s.byDir)
  const recs = recommendations(form, draft.values)
  const chosen = chosenDependencies(draft, form)
  const ids = [...new Set([...recs.map((r) => r.package), ...chosen])]
  usePackageDetails(game, profile, ids)
  // The rows and the search get stable handlers that call the latest, so their memo holds.
  const latest = useRef({ chosen, optional: draft.optional, set })
  latest.current = { chosen, optional: draft.optional, set }
  const toggle = useCallback((id: string, on: boolean) => {
    const { chosen: now, set: apply } = latest.current
    apply({ dependencies: on ? [...now, id] : now.filter((x) => x !== id) })
  }, [])
  const setOptional = useCallback((id: string, on: boolean) => {
    const { optional, set: apply } = latest.current
    apply({ optional: on ? [...optional, id] : optional.filter((x) => x !== id) })
  }, [])
  const add = useCallback((item: Item) => {
    knowPackage(item)
    const { chosen: now, set: apply } = latest.current
    apply({ dependencies: [...now, item.id] })
  }, [])
  return (
    <FormSection title={t`Starting dependencies`}>
      {ids.length === 0 ? (
        <Typography sx={{ color: 'text.secondary' }}>
          {t`Nothing to suggest from your answers; search for any mod yours builds on.`}
        </Typography>
      ) : null}
      {ids.map((id) => {
        const rec = recs.find((r) => r.package === id)
        return (
          <PackageRow
            key={id}
            id={id}
            item={known[id] ?? ownProject(projects, id)}
            why={rec ? t`Suggested for ${{ answer: lowerFirst(answerLabel(form, rec)) }}` : ''}
            checked={chosen.includes(id)}
            onToggle={toggle}
            optional={draft.optional.includes(id)}
            {...(draft.kind ? {} : { onOptional: setOptional })}
          />
        )
      })}
      <AddAnother game={game} profile={profile} taken={ids.join(',')} onAdd={add} />
    </FormSection>
  )
}
