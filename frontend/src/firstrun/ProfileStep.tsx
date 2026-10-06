import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, TextField, Typography } from '@mui/material'
import { Download, FolderInput, Link2, Plus } from 'lucide-react'
import { useEffect, useState } from 'react'
import { RegisterLinks } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nxmsvc/service.ts'
import { LocalProfiles } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import {
  Create,
  PreviewGameMods,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import {
  SetLastGame,
  SetLastProfile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useGameLoader, useGameName } from '../games/info.ts'
import { type GameId, useNav } from '../nav/store.ts'
import { useExternalImportSources } from '../profiles/externalImportSources.ts'
import { GameModsDialog } from '../profiles/GameModsDialog.tsx'
import { ImportWizard } from '../profiles/ImportWizard.tsx'
import { PackImportDialog } from '../profiles/PackImportDialog.tsx'
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

// Shown only when another mod manager on this computer has profiles for the game.
function ManagerImportCard({ game }: { game: GameId }) {
  const { t } = useLingui()
  const external = useExternalImportSources(game)
  const [packs, setPacks] = useState(0)
  const [wizard, setWizard] = useState(false)
  const [packPath, setPackPath] = useState<string | null>(null)
  useEffect(() => {
    LocalProfiles(game)
      .then((list) => setPacks(list?.length ?? 0))
      .catch(() => setPacks(0))
  }, [game])
  const found = external.reduce((n, s) => n + (s.profiles?.length ?? 0), packs)
  if (found === 0) {
    return null
  }
  return (
    <Box sx={cardSx('transparent')}>
      <Box sx={{ color: 'var(--mortar-accent-ink)', display: 'flex' }}>
        <Download size={28} />
      </Box>
      <Typography sx={{ fontSize: 18, fontWeight: 700 }}>{t`From another mod manager`}</Typography>
      <Typography sx={{ fontSize: 14, lineHeight: 1.5 }}>
        {plural(found, {
          one: '# profile from another mod manager is on this computer. Its mods download from their own sites.',
          other:
            '# profiles from other mod managers are on this computer. Their mods download from their own sites.',
        })}
      </Typography>
      <Box sx={{ flex: 1 }} />
      <Button variant="outlined" size="large" onClick={() => setWizard(true)}>
        {t`Choose a profile…`}
      </Button>
      <ImportWizard
        open={wizard}
        game={game}
        onClose={() => setWizard(false)}
        onPickPack={setPackPath}
        onOwnCode={null}
      />
      <PackImportDialog
        open={packPath !== null}
        game={game}
        initialPath={packPath ?? ''}
        onClose={() => setPackPath(null)}
      />
    </Box>
  )
}

export function ProfileStep({ game, site }: { game: GameId; site: string }) {
  const { t } = useLingui()
  const gameName = useGameName(game)
  const loader = useGameLoader(game)
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
          // Two to four cards: a fourth wraps into two rows rather than squeezing every card.
          gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))',
          gap: 2,
        }}
      >
        {gameMods ? (
          <Box sx={cardSx('primary.main')}>
            <Box sx={{ color: 'var(--mortar-accent-ink)', display: 'flex' }}>
              <FolderInput size={28} />
            </Box>
            <Typography sx={{ fontSize: 18, fontWeight: 700 }}>
              {t`Import from the game's Mods folder`}
            </Typography>
            <Typography sx={{ fontSize: 14, lineHeight: 1.5 }}>
              {t`Copy the mods already in ${gameName}'s Mods folder into a new profile. The game folder is not changed.`}
            </Typography>
            <Box sx={{ flex: 1 }} />
            <Button variant="contained" onClick={() => setImportOpen(true)} size="large">
              {t`Preview Mods folder import…`}
            </Button>
          </Box>
        ) : null}
        <ManagerImportCard game={game} />
        <Box
          component="form"
          onSubmit={(e) => {
            e.preventDefault()
            submit().catch(reportUnexpected)
          }}
          sx={cardSx(gameMods ? 'transparent' : 'primary.main')}
        >
          <Box sx={{ color: 'var(--mortar-accent-ink)', display: 'flex' }}>
            <Plus size={28} />
          </Box>
          <Typography sx={{ fontSize: 18, fontWeight: 700 }}>{t`Start empty`}</Typography>
          <Typography sx={{ fontSize: 14, lineHeight: 1.5 }}>
            {loader
              ? t`A profile with only ${loader}. Add mods from ${site} or from archives.`
              : t`An empty profile. Add mods from ${site} or from archives.`}
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
          <Box sx={{ color: 'var(--mortar-accent-ink)', display: 'flex' }}>
            <Link2 size={28} />
          </Box>
          <Typography sx={{ fontSize: 18, fontWeight: 700 }}>{t`From a shared link`}</Typography>
          <Typography sx={{ fontSize: 14, lineHeight: 1.5 }}>
            {t`Got a Mortar link? You see what it holds first. Nexus mods need your Nexus sign-in; Mortar asks then.`}
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
