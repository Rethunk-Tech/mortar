import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { FolderOpen, Package } from 'lucide-react'
import { memo, type RefObject, useCallback, useEffect, useMemo, useRef, useState } from 'react'
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
import { i18n } from '../i18n/index.ts'
import { TileGrid } from '../share/TileGrid.tsx'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { Field, FormDialog, TextInput } from '../shell/dialog/FormDialog.tsx'
import { useLoaded } from '../shell/useLoaded.ts'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { reloadGames, useGames } from './info.ts'

const CUSTOM_PREFIX = 'custom-'
const MAX_NAME = 60

// The loaders a custom game can use, each with the kind of game it suits; guessed says the game's files chose it.
// Memoised: typing the game's name leaves the list alone.
const LoaderChoice = memo(function LoaderChoiceView({
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
    <Field
      label={t`How it is modded`}
      hint={guessed ? t`Mortar chose this from the game's files.` : undefined}
    >
      <TileGrid
        label={t`How it is modded`}
        tiles={loaders.map((l) => ({
          id: l.id,
          icon: <Package size={32} />,
          name: l.name,
          hint: hints[l.id] ?? '',
        }))}
        lastUsed={null}
        selected={loader}
        onChoose={onLoader}
        padded={false}
      />
    </Field>
  )
})

/** Adds a game missing from the catalog: its name, its executable and how it is modded. Kept on this computer only. */
// The name keeps its own state, so typing redraws this field and not the dialog. The dialog reads the name from `value`
// and hears only when it fills or empties; `suggested` fills an empty name (the chosen program's own).
function NameField({
  value,
  suggested,
  onFilled,
}: {
  value: RefObject<string>
  suggested: string
  onFilled: (filled: boolean) => void
}) {
  const { t } = useLingui()
  const [name, setName] = useState('')
  const change = useCallback(
    (next: string) => {
      value.current = next
      setName(next)
      onFilled(next.trim() !== '')
    },
    [value, onFilled],
  )
  useEffect(() => {
    if (suggested && value.current === '') {
      change(suggested)
    }
  }, [suggested, value, change])
  return <TextInput label={t`Name`} value={name} onChange={change} maxLength={MAX_NAME} />
}

function RemoveGameConfirm({
  removing,
  onDone,
}: {
  removing: { id: string; name: string } | null
  onDone: () => void
}) {
  const { t } = useLingui()
  return (
    <ConfirmationDialog
      open={removing !== null}
      tone="danger"
      title={t`Remove ${{ name: removing?.name ?? '' }}?`}
      body={t`Mortar forgets this game. Its files stay where they are.`}
      confirmLabel={t`Remove`}
      onCancel={onDone}
      onConfirm={() => {
        onDone()
        if (removing) {
          RemoveCustomGame(removing.id)
            .then(reloadGames)
            .catch(reportError(t`Could not remove the game`))
        }
      }}
    />
  )
}

const addIssue = (hasExe: boolean, hasName: boolean): string => {
  if (!(hasExe || hasName)) {
    return i18n._(msg`Choose the game's program and give it a name first.`)
  }
  if (!hasExe) {
    return i18n._(msg`Choose the game's program first.`)
  }
  if (!hasName) {
    return i18n._(msg`Give the game a name first.`)
  }
  return i18n._(msg`Give the Thunderstore community first.`)
}

export function AddGameDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useLingui()
  const name = useRef('')
  const [hasName, setHasName] = useState(false)
  const [suggested, setSuggested] = useState('')
  // A new round remounts the name field empty after a game is added.
  const [round, setRound] = useState(0)
  const [exe, setExe] = useState('')
  const [loader, setLoader] = useState('bepinex5')
  const [community, setCommunity] = useState('')
  // The loader the game's own files point to, once a program is chosen.
  const [guessed, setGuessed] = useState('')
  const [busy, run] = usePending()
  const [removing, setRemoving] = useState<{ id: string; name: string } | null>(null)
  const { data: loaders } = useLoaded(() => CustomLoaders(), [], [])
  const games = useGames()
  const mine = useMemo(() => games.filter((g) => g.id.startsWith(CUSTOM_PREFIX)), [games])
  const add = () =>
    run(
      () =>
        AddCustomGame(name.current, exe, loader, loader === 'shimloader' ? community.trim() : '')
          .then((game) => SetGameFolder(game.id, game.folder))
          .then(() => {
            reloadGames()
            name.current = ''
            setHasName(false)
            setSuggested('')
            setRound((r) => r + 1)
            setExe('')
            setGuessed('')
            onClose()
          }),
      { errorTitle: t`Could not add the game` },
    )
  // Everything but the footer: the name filling or emptying (hasName) redraws the dialog's frame and footer, not this.
  const body = useMemo(() => {
    const choose = () =>
      PickExecutable(t`Choose the game's program`)
        .then(async (path) => {
          if (!path) {
            return
          }
          setExe(path)
          const guess = await GuessCustomGame(path)
          setSuggested(guess.name)
          setGuessed(guess.loader)
          if (guess.loader) {
            setLoader(guess.loader)
          }
        })
        .catch(reportError(t`Could not open the file dialog`))
    return (
      <>
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
        <NameField key={round} value={name} suggested={suggested} onFilled={setHasName} />
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
          <LoaderChoice
            loaders={loaders ?? []}
            loader={loader}
            guessed={guessed !== ''}
            onLoader={setLoader}
          />
        </Box>
        {loader === 'shimloader' ? (
          <TextInput
            label={t`${{ source: sourceLabel('thunderstore') }} community`}
            placeholder="voices-of-the-void"
            hint={t`The game's name in its ${{ source: sourceLabel('thunderstore') }} address (thunderstore.io/c/…). Mortar gets ${{ loader: SHIMLOADER }} and the game's mods from it. Choose the game's shipping program, in Binaries/Win64.`}
            value={community}
            onChange={setCommunity}
          />
        ) : null}
        {mine.length > 0 ? (
          <Box>
            <Typography
              sx={{ fontSize: 'base.fontSize', fontWeight: 700 }}
            >{t`Games you added`}</Typography>
            {mine.map((g) => (
              <Box key={g.id} sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
                <Typography sx={{ flex: 1, fontSize: 'medium.fontSize' }}>{g.name}</Typography>
                <Button
                  size="small"
                  color="error"
                  onClick={() => setRemoving({ id: g.id, name: g.name })}
                >
                  {t`Remove`}
                </Button>
              </Box>
            ))}
          </Box>
        ) : null}
      </>
    )
  }, [exe, loader, community, guessed, loaders, round, suggested, mine, t])
  const needsCommunity = loader === 'shimloader' && community.trim() === ''
  const ready = exe !== '' && hasName && !needsCommunity
  return (
    <FormDialog
      open={open}
      onClose={onClose}
      size="md"
      title={t`Add a game`}
      submitLabel={t`Add`}
      issue={ready ? '' : addIssue(exe !== '', hasName)}
      dirty={hasName || exe !== ''}
      busy={busy}
      onSubmit={add}
    >
      {body}
      <RemoveGameConfirm removing={removing} onDone={() => setRemoving(null)} />
    </FormDialog>
  )
}
