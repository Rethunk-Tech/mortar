import { Autocomplete, TextField } from '@mui/material'
import { useId } from 'react'
import type { Choice } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import { Field } from '../shell/dialog/FormDialog.tsx'
import { useSelectedFirst } from '../shell/selectedFirst.ts'
import { space } from '../theme/density.ts'

const CHIP_PX = 24

/** Picks several of a template's choices, as Browse's category filter does; value is the picked values. */
export function ChoicePicker({
  label,
  helperText,
  choices,
  value,
  onChange,
}: {
  label: string
  helperText?: string
  choices: Choice[]
  value: string[]
  onChange: (next: string[]) => void
}) {
  const id = useId()
  const listed = useSelectedFirst(choices, (c) => value.includes(c.value))
  return (
    <Field label={label} hint={helperText} htmlFor={id}>
      <Autocomplete
        multiple={true}
        autoHighlight={true}
        size="small"
        disableCloseOnSelect={true}
        slotProps={{ chip: { size: 'small' } }}
        // The text input beside the chips is taller than a small chip, which pushes the chips off centre; it takes
        // the chips' height so the field's own padding is even.
        sx={{
          // The theme holds small fields to one control's height; this one grows a row per line of chips.
          '& .MuiAutocomplete-inputRoot.MuiInputBase-root.MuiInputBase-sizeSmall': {
            alignItems: 'center',
            rowGap: '4px',
            height: 'auto',
            minHeight: space.control,
          },
          '& .MuiOutlinedInput-root.MuiInputBase-sizeSmall .MuiAutocomplete-input': {
            py: 0,
            height: CHIP_PX,
          },
          '& .MuiAutocomplete-tag': { my: 0 },
        }}
        options={listed.options}
        onOpen={listed.onOpen}
        value={choices.filter((c) => value.includes(c.value))}
        getOptionLabel={(c) => c.label}
        isOptionEqualToValue={(a, b) => a.value === b.value}
        onChange={(_, next) => onChange(next.map((c) => c.value))}
        renderInput={(params) => (
          <TextField
            id={id}
            fullWidth={true}
            size="small"
            slotProps={{
              inputLabel: params.slotProps.inputLabel,
              input: params.slotProps.input,
              htmlInput: params.slotProps.htmlInput,
            }}
          />
        )}
      />
    </Field>
  )
}
