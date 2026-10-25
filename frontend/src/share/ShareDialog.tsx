import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { modsLabel } from '../i18n/counts.ts'
import { i18n } from '../i18n/index.ts'
import { userModCount } from '../profiles/count.ts'
import { useProfiles } from '../profiles/store.ts'
import { AppDialog } from '../shell/dialog/AppDialog.tsx'
import { DestinationGrid } from './DestinationGrid.tsx'
import { DestinationPanel } from './DestinationPanel.tsx'
import { type ShownInfo, sharedMods } from './logic.ts'
import { shareDestinations } from './methods.ts'
import { MortarFooter } from './ShareFooter.tsx'
import { useShareBuild } from './useShareBuild.ts'

function shareMessage(info: ShownInfo, gameName: string): string {
  const count = sharedMods(info)
  return i18n._(
    msg`Try my Mortar profile "${{ name: info.name }}" for ${{ game: gameName }} (${plural({ count }, { one: '# mod', other: '# mods' })}): ${{ web: info.web }}`,
  )
}

export function ShareDialog() {
  const { t } = useLingui()
  const gameName = useProfiles((s) => s.game?.name ?? '')
  const built = useShareBuild()
  const { profileId, close, destination, setDestination, choose, lastUsed, thunderstore, info } =
    built
  const shareable = info?.count ?? 0
  const count = useProfiles((st) => {
    const p = st.profiles.find((x) => x.id === profileId)
    return p ? userModCount(p) : shareable
  })
  const entries = shareDestinations({
    thunderstore,
    count: shareable,
    leftOut: info?.leftOut.length ?? 0,
  })
  const names = {
    mortar: t`Mortar`,
    nexus: t`${{ source: sourceLabel('nexus') }}`,
    thunderstore: sourceLabel('thunderstore'),
    nearby: t`Nearby computer`,
    list: t`Mod list`,
  }
  const message = info ? shareMessage(info, gameName) : ''
  const back = destination ? () => setDestination(null) : undefined
  return (
    <AppDialog
      closeButton={true}
      open={profileId !== '' && info !== null}
      onClose={close}
      onBack={back}
      size="lg"
      // Only the mod list's text needs the room; every other destination is as tall as what it shows.
      height={destination === 'list' ? 'min(600px, calc(100% - 64px))' : undefined}
      flush={true}
      title={info ? t`Share ${{ name: info.name }}` : ''}
      subtitle={destination ? names[destination] : [gameName, modsLabel(count)].join(' · ')}
      footer={destination === 'mortar' ? <MortarFooter built={built} message={message} /> : null}
    >
      {info && destination ? (
        <Box
          sx={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0, px: 3, py: 2.5 }}
        >
          <DestinationPanel built={built} destination={destination} />
        </Box>
      ) : null}
      {info && !destination ? (
        <DestinationGrid entries={entries} lastUsed={lastUsed} onChoose={choose} />
      ) : null}
    </AppDialog>
  )
}
