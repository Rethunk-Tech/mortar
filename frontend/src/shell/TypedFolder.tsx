import { useLingui } from '@lingui/react/macro'
import { Box, Button, TextField } from '@mui/material'
import { useState } from 'react'

// The path field a folder picker falls back to where there is no native folder dialog (server mode).
export function TypedFolder({ label, onUse }: { label: string; onUse: (dir: string) => void }) {
  const { t } = useLingui()
  const [typed, setTyped] = useState('')
  return (
    <Box component="span" sx={{ display: 'flex', gap: 1, pt: 1 }}>
      <TextField
        size="small"
        fullWidth={true}
        value={typed}
        onChange={(e) => setTyped(e.target.value)}
        label={label}
      />
      <Button disabled={!typed.trim()} onClick={() => onUse(typed.trim())}>
        {t`Use folder`}
      </Button>
    </Box>
  )
}
