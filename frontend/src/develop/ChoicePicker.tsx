import { Autocomplete, TextField } from '@mui/material'
import { useId } from 'react'
import type { Choice } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import { Field } from '../shell/dialog/FormDialog.tsx'

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
  return (
    <Field label={label} hint={helperText} htmlFor={id}>
      <Autocomplete
        multiple={true}
        autoHighlight={true}
        size="small"
        slotProps={{ chip: { size: 'small' } }}
        options={choices}
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
