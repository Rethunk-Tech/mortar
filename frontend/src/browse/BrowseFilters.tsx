import { useLingui } from '@lingui/react/macro'
import { Autocomplete, Box, Chip, TextField } from '@mui/material'
import { memo, useLayoutEffect, useRef, useState } from 'react'
import { PrefSelect } from '../settings/PrefControls.tsx'
import { Tooltip } from '../shell/Tooltip.tsx'
import { space } from '../theme/density.ts'
import { BrowseShow } from './BrowseShow.tsx'
import type { BrowseModes } from './browseModes.ts'
import type { BrowseFilter } from './browseTypes.ts'
import { fitChips } from './fitChips.ts'

const MIN_PICKER_PX = 180
const CHIP_GAP_PX = 4
const MORE_CHIP_PX = 44
// Room the typing area keeps after the chips: Autocomplete's input minimum and its padding.
const INPUT_ROOM_PX = 48

// The chosen categories on one row. Chips that do not fit are replaced by a "+N" chip whose tooltip lists them. Every
// chip is also drawn once, invisibly and out of flow, to measure it, so the row never changes height or jumps.
function FittedChips({
  chips,
  getItemProps,
}: {
  chips: string[]
  getItemProps: (args: { index: number }) => Record<string, unknown> & { key?: unknown }
}) {
  const row = useRef<HTMLDivElement>(null)
  const measure = useRef<HTMLDivElement>(null)
  const [shown, setShown] = useState(chips.length)
  useLayoutEffect(() => {
    if (chips.length === 0) {
      return
    }
    // The row only takes the chips' width, so the caret sits right after them; what fits is measured on the field.
    const field = row.current?.parentElement
    const refit = () => {
      const widths = [...(measure.current?.children ?? [])].map(
        (c) => c.getBoundingClientRect().width,
      )
      const style = field ? getComputedStyle(field) : null
      const room =
        field && style
          ? field.clientWidth -
            Number.parseFloat(style.paddingLeft) -
            Number.parseFloat(style.paddingRight) -
            INPUT_ROOM_PX
          : 0
      setShown(fitChips(widths, Math.max(room, 0), CHIP_GAP_PX, MORE_CHIP_PX))
    }
    refit()
    const observer = new ResizeObserver(refit)
    if (field) {
      observer.observe(field)
    }
    return () => observer.disconnect()
  }, [chips])
  const hidden = chips.slice(shown)
  return (
    <Box
      ref={row}
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: `${CHIP_GAP_PX}px`,
        // Autocomplete's own margin on its chips would set them off from the +N chip, which is not one of its tags, and
        // its width cap is relative to this row, which is only as wide as the chips.
        '& .MuiAutocomplete-tag': { m: 0, maxWidth: 'none' },
        minWidth: 0,
        flex: '0 1 auto',
        overflow: 'hidden',
        position: 'relative',
      }}
    >
      <Box
        ref={measure}
        aria-hidden={true}
        sx={{
          position: 'absolute',
          visibility: 'hidden',
          display: 'flex',
          gap: `${CHIP_GAP_PX}px`,
          pointerEvents: 'none',
        }}
      >
        {chips.map((chip) => (
          <Chip
            key={chip}
            size="small"
            label={chip}
            onDelete={() => undefined}
            sx={{ flexShrink: 0 }}
          />
        ))}
      </Box>
      {chips.slice(0, shown).map((chip, index) => {
        const { key: itemKey, ...props } = getItemProps({ index })
        return (
          <Chip key={String(itemKey)} size="small" label={chip} sx={{ flexShrink: 0 }} {...props} />
        )
      })}
      {hidden.length > 0 ? (
        <Tooltip title={hidden.join(', ')}>
          <Chip size="small" label={`+${hidden.length}`} sx={{ flexShrink: 0 }} />
        </Tooltip>
      ) : null}
    </Box>
  )
}

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
      autoHighlight={true}
      size="small"
      options={options}
      value={value}
      disabled={disabled}
      onChange={(_, next) => onChange(next)}
      // The field keeps its one-row height: chips stay on a line and the ones past its edge are clipped.
      renderValue={(chips, getItemProps) => (
        <FittedChips chips={chips} getItemProps={getItemProps} />
      )}
      sx={{
        flex: 1,
        minWidth: MIN_PICKER_PX,
        '& .MuiAutocomplete-inputRoot': { flexWrap: 'nowrap' },
      }}
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
const BrowseFilters = memo(function BrowseFiltersView({
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
        vsmoddb: t`Sort: most followed`,
        paradox: t`Sort: top rated`,
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
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: space.gap,
        flexWrap: 'wrap',
        px: space.pad,
      }}
    >
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
})

export { BrowseFilters }
