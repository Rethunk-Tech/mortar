import { useLingui } from '@lingui/react/macro'
import { Box, Button, TextField, Typography } from '@mui/material'
import { FolderInput, Link2, Plus } from 'lucide-react'
import { useEffect, useState } from 'react'
import { RegisterLinks } from '../../bindings/github.com/Rethunk-AI/mortar/internal/nxmsvc/service.ts'
import {
  Create,
  PreviewGameMods,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import {
  SetLastGame,
  SetLastProfile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { type GameId, useNav } from '../nav/store.ts'
import { GameModsDialog } from '../profiles/GameModsDialog.tsx'
import { openImport } from '../share/store.ts'
import { errorMessage, reportUnexpected } from '../toasts/report.ts'

const cardSx = (borderColor: string) => ({
  display: 'flex',
  flexDirection: 'column',
  gap: '10px',
  p: '22px',
  bgcolor: 'var(--mortar-paper-78)',
  border: '2px solid',
  borderColor,
  borderRadius: '8px',
})

export function ProfileStep({ game }: { game: GameId }) {
  const { t } = useLingui()
  const [name, setName] = useState(t`Main`)
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [link, setLink] = useState('')
  const [gameMods, setGameMods] = useState(false)
  const [importOpen, setImportOpen] = useState(false)
  useEffect(() => {
    PreviewGameMods(game)
      .then((p) => setGameMods((p.mods ?? []).length > 0))
      .catch(() => setGameMods(false))
  }, [game])
  const submit = async () => {
    const trimmed = name.trim()
    if (!trimmed) {
      setError(t`Enter a name for the profile.`)
      return
    }
    setBusy(true)
    try {
      const profile = await Create(game, trimmed)
      await Promise.all([SetLastGame(game), SetLastProfile(game, profile.id)])
      RegisterLinks().catch(reportUnexpected)
      useNav.getState().openGame(game)
    } catch (e) {
      setError(errorMessage(e))
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
      <Box
        sx={{
          display: 'grid',
          gridTemplateColumns: gameMods ? 'repeat(3, minmax(0, 1fr))' : 'repeat(2, minmax(0, 1fr))',
          gap: 2,
        }}
      >
        {gameMods ? (
          <Box sx={cardSx('primary.main')}>
            <Box sx={{ color: 'primary.main', display: 'flex' }}>
              <FolderInput size={28} />
            </Box>
            <Typography sx={{ fontSize: 18, fontWeight: 700 }}>
              {t`Import from the game's Mods folder`}
            </Typography>
            <Typography sx={{ fontSize: 14, lineHeight: 1.5 }}>
              {t`Copy the mods already in Stardew Valley's Mods folder into a new profile. Nothing in the game folder is moved or changed.`}
            </Typography>
            <Box sx={{ flex: 1 }} />
            <Button variant="contained" onClick={() => setImportOpen(true)} size="large">
              {t`Preview Mods folder import…`}
            </Button>
          </Box>
        ) : null}
        <Box
          component="form"
          onSubmit={(e) => {
            e.preventDefault()
            submit().catch(reportUnexpected)
          }}
          sx={cardSx(gameMods ? 'transparent' : 'primary.main')}
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
            variant={gameMods ? 'outlined' : 'contained'}
            disabled={busy}
            size="large"
          >
            {t`New profile`}
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
          <Button type="submit" variant="outlined" size="large">
            {t`Preview link…`}
          </Button>
        </Box>
      </Box>
      <GameModsDialog
        open={importOpen}
        game={game}
        onClose={() => setImportOpen(false)}
        onImported={(id) => {
          Promise.all([SetLastGame(game), SetLastProfile(game, id)])
            .then(() => useNav.getState().openGame(game))
            .catch(reportUnexpected)
          RegisterLinks().catch(reportUnexpected)
        }}
      />
    </Box>
  )
}
