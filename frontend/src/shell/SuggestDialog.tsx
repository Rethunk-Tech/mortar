import { useLingui } from '@lingui/react/macro'
import { DiscordLogo } from '../brand/Discord.tsx'
import { SourceLogo } from '../brand/sources/SourceLogo.tsx'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { openPage } from '../mods/menu.ts'
import { type Tile, TileGrid } from '../share/TileGrid.tsx'
import { type Destination, GO, getDestination, SUGGEST_KEY, setDestination } from './destination.ts'
import { AppDialog } from './dialog/AppDialog.tsx'
import { useSuggest } from './suggest.ts'

const ROUTES: Record<Destination, string> = {
  discord: `${GO}suggestions/`,
  github: 'https://github.com/Rethunk-Tech/mortar/issues/new?template=feature_request.yml',
}

const LOGO_PX = 32
const GITHUB: Destination = 'github'

// Two places to post, side by side; the one used last takes focus.
export function SuggestDialog() {
  const { t } = useLingui()
  const open = useSuggest((s) => s.open)
  const close = () => useSuggest.setState({ open: false })
  // Read on each open, so the place used last takes focus.
  const last = open ? getDestination(SUGGEST_KEY) : 'github'
  const go = (to: Destination) => {
    setDestination(SUGGEST_KEY, to)
    close()
    openPage(ROUTES[to])
  }
  const tiles: Tile<Destination>[] = [
    {
      id: 'discord',
      icon: <DiscordLogo size={LOGO_PX} />,
      name: sourceLabel('discord'),
      hint: t`Others can vote on it`,
    },
    {
      id: 'github',
      icon: <SourceLogo id={GITHUB} size={LOGO_PX} />,
      name: sourceLabel('github'),
      hint: t`Kept next to the code`,
    },
  ]
  return (
    <AppDialog
      open={open}
      onClose={close}
      title={t`Suggest a feature`}
      subtitle={t`Where should your idea go?`}
      size="md"
      flush={true}
    >
      <TileGrid label={t`Where to post`} tiles={tiles} lastUsed={last} onChoose={go} />
    </AppDialog>
  )
}
