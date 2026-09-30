import { useLingui } from '@lingui/react/macro'
import { Box, Button, TextField, Typography } from '@mui/material'
import { Plus } from 'lucide-react'
import { useState } from 'react'
import { Create } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import {
  SetLastGame,
  SetLastProfile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useNav } from '../nav/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { STARDEW } from './needed.ts'

export function ProfileStep() {
  const { t } = useLingui()
  const [name, setName] = useState('Main')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const submit = async () => {
    const trimmed = name.trim()
    if (!trimmed) {
      setError(t`Enter a name for the profile.`)
      return
    }
    setBusy(true)
    try {
      const profile = await Create(STARDEW, trimmed)
      await Promise.all([SetLastGame(STARDEW), SetLastProfile(STARDEW, profile.id)])
      useNav.getState().openGame(STARDEW)
    } catch (e) {
      setError(String(e))
      setBusy(false)
    }
  }
  return (
    <Box
      component="form"
      onSubmit={(e) => {
        e.preventDefault()
        submit().catch(reportUnexpected)
      }}
      sx={{
        width: 'min(420px, calc(100% - 32px))',
        display: 'flex',
        flexDirection: 'column',
        gap: 2,
      }}
    >
      <Typography sx={{ textAlign: 'center', fontSize: 24, fontWeight: 700 }}>
        {t`Make your first profile`}
      </Typography>
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: '10px',
          p: '22px',
          bgcolor: 'rgba(50,50,60,0.78)',
          border: '2px solid',
          borderColor: 'primary.main',
          borderRadius: '8px',
        }}
      >
        <Box sx={{ color: 'primary.main', display: 'flex' }}>
          <Plus size={28} />
        </Box>
        <Typography sx={{ fontSize: 18, fontWeight: 700 }}>{t`Start empty`}</Typography>
        <Typography sx={{ fontSize: 14, lineHeight: 1.5 }}>
          {t`A profile with only SMAPI. Add mods from Nexus or from archives.`}
        </Typography>
        <TextField
          autoFocus={true}
          fullWidth={true}
          size="small"
          label={t`Name`}
          value={name}
          onChange={(e) => {
            setName(e.target.value)
            setError('')
          }}
          error={error !== ''}
          helperText={error || ' '}
          slotProps={{ htmlInput: { maxLength: 60 }, root: { sx: { userSelect: 'text' } } }}
        />
      </Box>
      <Button
        type="submit"
        variant="contained"
        disabled={busy}
        sx={{ height: 46, fontSize: 16, fontWeight: 700, whiteSpace: 'nowrap' }}
      >
        {t`Create profile`}
      </Button>
    </Box>
  )
}
