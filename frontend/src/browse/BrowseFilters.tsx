import { useLingui } from '@lingui/react/macro'
import { Autocomplete, Box, TextField } from '@mui/material'
import { PrefSelect } from '../settings/PrefControls.tsx'
import { BrowseShow } from './BrowseShow.tsx'
import type { BrowseModes } from './browseModes.ts'
import type { BrowseFilter } from './browseTypes.ts'

const MIN_PICKER_PX = 180

function CategoryPicker({
  label,
  options,
  value,
  onChange,
  disabled,
  title,
}: {
  label: string
  options: string[]
  value: string[]
  onChange: (next: string[]) => void
  disabled: boolean
  title: string
}) {
  return (
    <Autocomplete
      multiple={true}
      size="small"
      options={options}
      value={value}
      disabled={disabled}
      onChange={(_, next) => onChange(next)}
      slotProps={{ chip: { size: 'small' } }}
      sx={{ flex: 1, minWidth: MIN_PICKER_PX }}
      renderInput={(params) => (
        <TextField
          id={params.id}
          disabled={params.disabled}
          fullWidth={params.fullWidth}
          size="small"
          label={label}
          title={disabled ? title : undefined}
          slotProps={{
            inputLabel: params.slotProps.inputLabel,
            input: params.slotProps.input,
            htmlInput: params.slotProps.htmlInput,
          }}
        />
      )}
    />
  )
}

// One compact row: include and exclude category pickers, then the sort. The pickers are disabled for a source with
// no categories (GitHub, or Nexus while signed out).
function BrowseFilters({
  categories,
  filter,
  onFilter,
  modes,
  onModes,
  hasCompat,
  source,
  sorts,
}: {
  categories: string[]
  filter: BrowseFilter
  onFilter: (next: BrowseFilter) => void
  modes: BrowseModes
  onModes: (next: BrowseModes) => void
  hasCompat: boolean
  source: string
  sorts: string[]
}) {
  const { t } = useLingui()
  // The metric is named per source, since "most endorsed" on Nexus is "most followed" on Modrinth.
  const labels: Record<string, string> = {
    downloads: t`Sort: most downloaded`,
    endorsements:
      {
        nexus: t`Sort: most endorsed`,
        thunderstore: t`Sort: top rated`,
        modrinth: t`Sort: most followed`,
        curseforge: t`Sort: most popular`,
      }[source] ?? t`Sort: most endorsed`,
    stars: t`Sort: most stars`,
    forks: t`Sort: most forks`,
    updated: t`Sort: recently updated`,
    newest: t`Sort: newest`,
    name: t`Sort: name (A to Z)`,
  }
  const none = categories.length === 0
  const title = t`This source has no categories to filter by`
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap', px: 2, pb: 0.75 }}>
      <CategoryPicker
        label={t`Include categories`}
        options={categories}
        value={filter.include}
        onChange={(include) => onFilter({ ...filter, include })}
        disabled={none}
        title={title}
      />
      <CategoryPicker
        label={t`Exclude categories`}
        options={categories}
        value={filter.exclude}
        onChange={(exclude) => onFilter({ ...filter, exclude })}
        disabled={none}
        title={title}
      />
      <BrowseShow modes={modes} onModes={onModes} hasCompat={hasCompat} />
      <PrefSelect
        label={t`Sort`}
        value={filter.sort}
        onChange={(sort) => onFilter({ ...filter, sort })}
        options={[
          { value: '', label: t`Sort: best match` },
          ...sorts.map((value) => ({ value, label: labels[value] ?? value })),
        ]}
      />
    </Box>
  )
}

export { BrowseFilters }
