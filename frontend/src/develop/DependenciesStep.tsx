import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import { Search } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/browse/service.ts'
import type { Form } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import type { Item } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/source/models.ts'
import { LetterTile } from '../mods/LetterTile.tsx'
import { FormSection } from '../shell/dialog/FormDialog.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { space } from '../theme/density.ts'
import { findPackage, type Known, NO_FILTER, THUNDERSTORE } from './dependencies.ts'
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

function PackageRow({
  id,
  item,
  why,
  checked,
  onToggle,
}: {
  id: string
  item: Item | undefined
  why: string
  checked: boolean
  onToggle: (on: boolean) => void
}) {
  const name = item?.name ?? id
  return (
    <Box
      component="label"
      sx={{ display: 'flex', alignItems: 'center', gap: space.gap, cursor: 'pointer' }}
    >
      <Checkbox checked={checked} onChange={(_, on) => onToggle(on)} />
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
    </Box>
  )
}

function useSearch(game: string, profile: string, query: string) {
  const [hits, setHits] = useState<Item[]>([])
  useEffect(() => {
    if (query.trim().length < 2) {
      setHits([])
      return
    }
    const timer = setTimeout(() => {
      Search(game, THUNDERSTORE, query, 1, profile, NO_FILTER)
        .then((page) => setHits((page.items ?? []).slice(0, SEARCH_HITS)))
        .catch(() => setHits([]))
    }, SEARCH_DELAY_MS)
    return () => clearTimeout(timer)
  }, [game, profile, query])
  return hits
}

/** The packages the mod starts depending on: those the answers suggest, ticked, and any the player searches for. */
export function DependenciesStep({
  game,
  profile,
  form,
  draft,
  known,
  setKnown,
  set,
}: {
  game: string
  profile: string
  form: Form | null
  draft: Draft
  known: Known
  setKnown: (update: (k: Known) => Known) => void
  set: (next: Partial<Draft>) => void
}) {
  const { t } = useLingui()
  const [query, setQuery] = useState('')
  const recs = recommendations(form, draft.values)
  const chosen = chosenDependencies(draft, form)
  const ids = [...new Set([...recs.map((r) => r.package), ...chosen])]
  const missing = ids.filter((id) => !known[id]).join(',')
  useEffect(() => {
    for (const id of missing.split(',').filter(Boolean)) {
      findPackage(game, profile, id)
        .then((item) => item && setKnown((k) => ({ ...k, [id]: item })))
        .catch(() => undefined)
    }
  }, [game, profile, missing, setKnown])
  const hits = useSearch(game, profile, query).filter((h) => !ids.includes(h.id))
  const toggle = (id: string, on: boolean) =>
    set({ dependencies: on ? [...chosen, id] : chosen.filter((x) => x !== id) })
  return (
    <>
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
              item={known[id]}
              why={rec ? t`Suggested for ${{ answer: answerLabel(form, rec) }}` : ''}
              checked={chosen.includes(id)}
              onToggle={(on) => toggle(id, on)}
            />
          )
        })}
      </FormSection>
      <FormSection title={t`Add another`}>
        <SearchField
          value={query}
          onChange={setQuery}
          label={t`Search mods`}
          placeholder={t`Search mods`}
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
              <Typography
                noWrap={true}
                sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}
              >
                {h.author}
              </Typography>
            </Box>
            <Button
              size="small"
              variant="outlined"
              onClick={() => {
                setKnown((k) => ({ ...k, [h.id]: h }))
                set({ dependencies: [...chosen, h.id] })
              }}
            >
              {t`Add`}
            </Button>
          </Box>
        ))}
      </FormSection>
    </>
  )
}
