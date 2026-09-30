import { useLingui } from '@lingui/react/macro'
import { Box, Button, TextField, Typography } from '@mui/material'
import { Link2, Plus } from 'lucide-react'
import { useState } from 'react'
import { RegisterLinks } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nxmsvc/service.ts'
import { Create } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import {
  SetLastGame,
  SetLastProfile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useNav } from '../nav/store.ts'
import { openImport } from '../share/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { STARDEW } from './needed.ts'

const cardSx = (borderColor: string) => ({
  display: 'flex',
  flexDirection: 'column',
  gap: '10px',
  p: '22px',
  bgcolor: 'rgba(50,50,60,0.78)',
  border: '2px solid',
  borderColor,
  borderRadius: '8px',
})

export function ProfileStep() {
  const { t } = useLingui()
  const [name, setName] = useState('Main')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [link, setLink] = useState('')
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
      RegisterLinks().catch(reportUnexpected)
      useNav.getState().openGame(STARDEW)
    } catch (e) {
      setError(String(e))
      setBusy(false)
    }
  }
  return (
    <Box
      sx={{
        width: 'min(880px, calc(100% - 32px))',
        display: 'flex',
        flexDirection: 'column',
        gap: 2,
      }}
    >
      <Typography sx={{ textAlign: 'center', fontSize: 24, fontWeight: 700 }}>
        {t`Make your first profile`}
      </Typography>
      <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, minmax(0, 1fr))', gap: 2 }}>
        <Box
          component="form"
          onSubmit={(e) => {
            e.preventDefault()
            submit().catch(reportUnexpected)
          }}
          sx={cardSx('primary.main')}
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
          <Button
            type="submit"
            variant="contained"
            disabled={busy}
            sx={{ height: 46, fontSize: 16, fontWeight: 700, whiteSpace: 'nowrap' }}
          >
            {t`Create profile`}
          </Button>
        </Box>
        <Box
          component="form"
          onSubmit={(e) => {
            e.preventDefault()
            openImport({ link: link.trim() })
          }}
          sx={cardSx('transparent')}
        >
          <Box sx={{ color: 'primary.main', display: 'flex' }}>
            <Link2 size={28} />
          </Box>
          <Typography sx={{ fontSize: 18, fontWeight: 700 }}>{t`From a shared link`}</Typography>
          <Typography sx={{ fontSize: 14, lineHeight: 1.5 }}>
            {t`Someone sent you a Mortar link? You see what it holds first. Mods from Nexus need your Nexus sign-in, which Mortar asks for then.`}
          </Typography>
          <TextField
            fullWidth={true}
            size="small"
            label={t`Share link`}
            value={link}
            onChange={(e) => setLink(e.target.value)}
            helperText=" "
            slotProps={{ root: { sx: { userSelect: 'text' } } }}
          />
          <Button
            type="submit"
            variant="outlined"
            sx={{ height: 46, fontSize: 16, fontWeight: 700, whiteSpace: 'nowrap' }}
          >
            {t`Preview import`}
          </Button>
        </Box>
      </Box>
    </Box>
  )
}
