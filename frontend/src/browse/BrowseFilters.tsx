import { useLingui } from '@lingui/react/macro'
import { Autocomplete, Box, TextField } from '@mui/material'
import { PrefSelect } from '../settings/PrefControls.tsx'
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
}: {
  categories: string[]
  filter: BrowseFilter
  onFilter: (next: BrowseFilter) => void
}) {
  const { t } = useLingui()
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
      <PrefSelect
        label={t`Sort`}
        value={filter.sort}
        onChange={(sort) => onFilter({ ...filter, sort })}
        options={[
          { value: '', label: t`Sort: best match` },
          { value: 'downloads', label: t`Sort: most downloaded` },
          { value: 'endorsements', label: t`Sort: most endorsed` },
          { value: 'updated', label: t`Sort: recently updated` },
          { value: 'name', label: t`Sort: name` },
        ]}
      />
    </Box>
  )
}

export { BrowseFilters }
