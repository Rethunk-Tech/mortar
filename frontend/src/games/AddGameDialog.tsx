import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Radio,
  RadioGroup,
  TextField,
  Typography,
} from '@mui/material'
import { useState } from 'react'
import {
  AddCustomGame,
  CustomLoaders,
  RemoveCustomGame,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import { PickExecutable } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { SetGameFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'
import { reloadGames, useGames } from './info.ts'

const CUSTOM_PREFIX = 'custom-'

/** Adds a game missing from the catalog: its name, its executable and how it is modded. Kept on this computer only. */
export function AddGameDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useLingui()
  const [name, setName] = useState('')
  const [exe, setExe] = useState('')
  const [loader, setLoader] = useState('bepinex5')
  const [community, setCommunity] = useState('')
  const [busy, setBusy] = useState(false)
  const { data: loaders } = useLoaded(() => CustomLoaders(), [], [])
  const mine = useGames().filter((g) => g.id.startsWith(CUSTOM_PREFIX))
  const add = () => {
    setBusy(true)
    AddCustomGame(name, exe, loader, loader === 'shimloader' ? community.trim() : '')
      .then((game) => SetGameFolder(game.id, game.folder))
      .then(() => {
        reloadGames()
        setName('')
        setExe('')
        onClose()
      })
      .catch(reportError(t`Could not add the game`))
      .finally(() => setBusy(false))
  }
  const remove = (id: string) =>
    RemoveCustomGame(id).then(reloadGames).catch(reportError(t`Could not remove the game`))
  return (
    <Dialog open={open} onClose={busy ? undefined : onClose} fullWidth={true} maxWidth="sm">
      <DialogTitle>{t`Add a game`}</DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap, pt: 0.5 }}>
          <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
            {loader === 'shimloader'
              ? t`For a game that is not in Mortar's list. It stays on this computer, and its mods come from its Thunderstore community.`
              : t`For a game that is not in Mortar's list. It stays on this computer, and its mods come from archives you add, since Mortar knows no site for it.`}
          </Typography>
          <TextField
            size="small"
            label={t`Name`}
            value={name}
            onChange={(e) => setName(e.target.value)}
            slotProps={{ htmlInput: { maxLength: 60 } }}
          />
          <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
            <Typography sx={{ flex: 1, fontSize: 13, overflowWrap: 'anywhere' }}>
              {exe || t`No game program chosen`}
            </Typography>
            <Button
              size="small"
              onClick={() =>
                PickExecutable(t`Choose the game's program`)
                  .then((path) => path && setExe(path))
                  .catch(reportError(t`Could not choose the program`))
              }
            >
              {t`Choose the game's program…`}
            </Button>
          </Box>
          <Typography sx={{ fontSize: 13, fontWeight: 700 }}>{t`How it is modded`}</Typography>
          <RadioGroup value={loader} onChange={(_, v) => setLoader(v)}>
            {(loaders ?? []).map((l) => (
              <FormControlLabel
                key={l.id}
                value={l.id}
                control={<Radio size="small" />}
                label={l.name}
              />
            ))}
          </RadioGroup>
          {loader === 'shimloader' ? (
            <TextField
              size="small"
              label={t`Thunderstore community`}
              placeholder="voices-of-the-void"
              helperText={t`The game's name in its Thunderstore address (thunderstore.io/c/…). Mortar gets Shimloader and the game's mods from it. Choose the game's shipping program, in Binaries/Win64.`}
              value={community}
              onChange={(e) => setCommunity(e.target.value)}
            />
          ) : null}
          {mine.length > 0 ? (
            <Box>
              <Typography sx={{ fontSize: 13, fontWeight: 700 }}>{t`Games you added`}</Typography>
              {mine.map((g) => (
                <Box key={g.id} sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
                  <Typography sx={{ flex: 1, fontSize: 14 }}>{g.name}</Typography>
                  <Button size="small" color="error" onClick={() => remove(g.id)}>
                    {t`Remove`}
                  </Button>
                </Box>
              ))}
            </Box>
          ) : null}
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={busy}>
          {t`Close`}
        </Button>
        <Button
          variant="contained"
          disabled={
            busy ||
            name.trim() === '' ||
            exe === '' ||
            (loader === 'shimloader' && community.trim() === '')
          }
          onClick={add}
        >
          {t`Add`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
