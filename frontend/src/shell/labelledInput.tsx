import { type AutocompleteRenderInputParams, TextField } from '@mui/material'

/** An Autocomplete's renderInput with no visible label: the input is named by `label` for screen readers. */
export const labelledInput =
  (label: string, placeholder?: string) => (params: AutocompleteRenderInputParams) => (
    <TextField
      {...params}
      {...(placeholder === undefined ? {} : { placeholder })}
      slotProps={{
        ...params.slotProps,
        htmlInput: { ...params.slotProps.htmlInput, 'aria-label': label },
      }}
    />
  )
