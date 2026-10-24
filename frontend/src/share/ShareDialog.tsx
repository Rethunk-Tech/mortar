import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Dialog, DialogTitle, Typography } from '@mui/material'
import { ArrowLeft, X } from 'lucide-react'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { modsLabel } from '../i18n/counts.ts'
import { i18n } from '../i18n/index.ts'
import { userModCount } from '../profiles/count.ts'
import { useProfiles } from '../profiles/store.ts'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { space } from '../theme/density.ts'
import { DestinationGrid } from './DestinationGrid.tsx'
import { DestinationPanel } from './DestinationPanel.tsx'
import { type ShownInfo, sharedMods } from './logic.ts'
import { shareDestinations } from './methods.ts'
import { MessageFooter } from './ShareFooter.tsx'
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
  return (
    <Dialog
      open={profileId !== '' && info !== null}
      onClose={(_, reason) => {
        if (reason === 'escapeKeyDown' && destination) {
          setDestination(null)
          return
        }
        close()
      }}
      maxWidth="md"
      fullWidth={true}
      slotProps={{
        paper: {
          sx: {
            bgcolor: 'var(--mortar-panel-solid)',
            border: '1px solid var(--mortar-hairline-12)',
            // Only the mod list's text needs the room; every other destination is as tall as what it shows.
            height: destination === 'list' ? 'min(600px, calc(100% - 64px))' : 'auto',
            overflow: 'hidden',
            borderRadius: '12px',
          },
        },
      }}
    >
      {info ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0 }}>
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: space.gap,
              pr: space.pad,
              pb: space.pad,
            }}
          >
            {destination ? (
              <Box sx={{ display: 'flex', pl: space.pad }}>
                <TipIconButton label={t`Back`} onClick={() => setDestination(null)}>
                  <ArrowLeft size={18} />
                </TipIconButton>
              </Box>
            ) : null}
            <Box sx={{ minWidth: 0, flex: 1 }}>
              <DialogTitle
                noWrap={true}
                title={info.name}
                sx={{ fontSize: 20, fontWeight: 700, ...(destination ? { pl: 0 } : {}) }}
              >
                {t`Share ${{ name: info.name }}`}
              </DialogTitle>
              <Typography
                noWrap={true}
                sx={{
                  color: 'text.secondary',
                  fontSize: 14,
                  px: space.pad,
                  ...(destination ? { pl: 0 } : {}),
                }}
              >
                {destination ? names[destination] : `${gameName} · ${modsLabel(count)}`}
              </Typography>
            </Box>
            <TipIconButton label={t`Close`} onClick={close}>
              <X size={18} />
            </TipIconButton>
          </Box>
          <Box
            sx={{
              flex: 1,
              display: 'flex',
              flexDirection: 'column',
              minHeight: 0,
              overflowY: 'auto',
              borderTop: '1px solid var(--mortar-hairline-muted)',
            }}
          >
            {destination ? (
              <Box sx={{ display: 'flex', flexDirection: 'column', flex: 1, p: '24px 28px' }}>
                <DestinationPanel built={built} destination={destination} />
              </Box>
            ) : (
              <DestinationGrid entries={entries} lastUsed={lastUsed} onChoose={choose} />
            )}
          </Box>
          {destination === 'mortar' ? <MessageFooter info={info} message={message} /> : null}
        </Box>
      ) : null}
    </Dialog>
  )
}
