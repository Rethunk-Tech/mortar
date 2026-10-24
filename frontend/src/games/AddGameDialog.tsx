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
import { FolderOpen } from 'lucide-react'
import { useState } from 'react'
import {
  AddCustomGame,
  CustomLoaders,
  GuessCustomGame,
  RemoveCustomGame,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/game/service.ts'
import { PickExecutable } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { SetGameFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { SHIMLOADER, UNITY } from '../brand/productNames.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { reloadGames, useGames } from './info.ts'

const CUSTOM_PREFIX = 'custom-'

// The loaders a custom game can use, each with the kind of game it suits; guessed says the game's files chose it.
function LoaderChoice({
  loaders,
  loader,
  guessed,
  onLoader,
}: {
  loaders: { id: string; name: string }[]
  loader: string
  guessed: boolean
  onLoader: (id: string) => void
}) {
  const { t } = useLingui()
  const hints: Record<string, string> = {
    bepinex5: t`${{ app: UNITY }} games with ${{ path: '*_Data/Managed' }} beside the game`,
    bepinex6: t`${{ app: UNITY }} games with ${{ path: 'GameAssembly.dll' }} beside the game`,
    melonloader: t`${{ app: UNITY }} games whose mods are made for it`,
    shimloader: t`Unreal Engine games, with their mods from ${{ source: sourceLabel('thunderstore') }}`,
    folder: t`Any game whose mods are files in a Mods folder`,
  }
  return (
    <>
      <Box>
        <Typography sx={{ fontSize: 'base.fontSize', fontWeight: 700 }}>
          {t`How it is modded`}
        </Typography>
        {guessed ? (
          <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
            {t`Mortar chose this from the game's files.`}
          </Typography>
        ) : null}
      </Box>
      <RadioGroup value={loader} onChange={(_, v) => onLoader(v)}>
        {loaders.map((l) => (
          <FormControlLabel
            key={l.id}
            value={l.id}
            control={<Radio size="small" />}
            sx={{ alignItems: 'flex-start', '& .MuiRadio-root': { mt: -0.5 } }}
            label={
              <Box>
                <Typography sx={{ fontSize: 'base.fontSize' }}>{l.name}</Typography>
                {hints[l.id] ? (
                  <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
                    {hints[l.id]}
                  </Typography>
                ) : null}
              </Box>
            }
          />
        ))}
      </RadioGroup>
    </>
  )
}

/** Adds a game missing from the catalog: its name, its executable and how it is modded. Kept on this computer only. */
export function AddGameDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useLingui()
  const [name, setName] = useState('')
  const [exe, setExe] = useState('')
  const [loader, setLoader] = useState('bepinex5')
  const [community, setCommunity] = useState('')
  // The loader the game's own files point to, once a program is chosen.
  const [guessed, setGuessed] = useState('')
  const [busy, run] = usePending()
  const { data: loaders } = useLoaded(() => CustomLoaders(), [], [])
  const mine = useGames().filter((g) => g.id.startsWith(CUSTOM_PREFIX))
  const add = () =>
    run(
      () =>
        AddCustomGame(name, exe, loader, loader === 'shimloader' ? community.trim() : '')
          .then((game) => SetGameFolder(game.id, game.folder))
          .then(() => {
            reloadGames()
            setName('')
            setExe('')
            setGuessed('')
            onClose()
          }),
      { errorTitle: t`Could not add the game` },
    )
  const choose = () =>
    PickExecutable(t`Choose the game's program`)
      .then(async (path) => {
        if (!path) {
          return
        }
        setExe(path)
        const guess = await GuessCustomGame(path)
        setName((current) => current || guess.name)
        setGuessed(guess.loader)
        if (guess.loader) {
          setLoader(guess.loader)
        }
      })
      .catch(reportError(t`Could not open the file dialog`))
  const ready =
    exe !== '' && name.trim() !== '' && (loader !== 'shimloader' || community.trim() !== '')
  const remove = (id: string) =>
    RemoveCustomGame(id).then(reloadGames).catch(reportError(t`Could not remove the game`))
  return (
    <Dialog open={open} onClose={busy ? undefined : onClose} fullWidth={true} maxWidth="sm">
      <DialogTitle>{t`Add a game`}</DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap, pt: 0.5 }}>
          <Typography sx={{ fontSize: 'base.fontSize', color: 'text.secondary' }}>
            {loader === 'shimloader'
              ? t`For a game that is not in Mortar's list. It stays on this computer, and its mods come from its ${{ source: sourceLabel('thunderstore') }} community.`
              : t`For a game that is not in Mortar's list. It stays on this computer, and its mods come from archives you add, since Mortar knows no site for it.`}
          </Typography>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
            <Button variant="outlined" startIcon={<FolderOpen size={16} />} onClick={choose}>
              {t`Choose the game's program…`}
            </Button>
            {exe ? (
              <Typography
                title={exe}
                sx={{
                  flex: 1,
                  minWidth: 0,
                  fontSize: 'medium.fontSize',
                  color: 'text.secondary',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                  direction: 'rtl',
                  textAlign: 'left',
                }}
              >
                {exe}
              </Typography>
            ) : null}
          </Box>
          <TextField
            size="small"
            label={t`Name`}
            value={name}
            onChange={(e) => setName(e.target.value)}
            slotProps={{ htmlInput: { maxLength: 60 } }}
          />
          <LoaderChoice
            loaders={loaders ?? []}
            loader={loader}
            guessed={guessed !== ''}
            onLoader={setLoader}
          />
          {loader === 'shimloader' ? (
            <TextField
              size="small"
              label={t`${{ source: sourceLabel('thunderstore') }} community`}
              placeholder="voices-of-the-void"
              helperText={t`The game's name in its ${{ source: sourceLabel('thunderstore') }} address (thunderstore.io/c/…). Mortar gets ${{ loader: SHIMLOADER }} and the game's mods from it. Choose the game's shipping program, in Binaries/Win64.`}
              value={community}
              onChange={(e) => setCommunity(e.target.value)}
            />
          ) : null}
          {ready ? null : (
            <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
              {t`Choose the game's program and give it a name first.`}
            </Typography>
          )}
          {mine.length > 0 ? (
            <Box>
              <Typography
                sx={{ fontSize: 'base.fontSize', fontWeight: 700 }}
              >{t`Games you added`}</Typography>
              {mine.map((g) => (
                <Box key={g.id} sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
                  <Typography sx={{ flex: 1, fontSize: 'medium.fontSize' }}>{g.name}</Typography>
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
        <Button variant="contained" disabled={busy || !ready} onClick={add}>
          {t`Add`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
