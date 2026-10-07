import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Dialog, Typography } from '@mui/material'
import { ArrowLeft, X } from 'lucide-react'
import { userModCount } from '../profiles/count.ts'
import { useProfiles } from '../profiles/store.ts'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { DestinationGrid } from './DestinationGrid.tsx'
import { DestinationPanel } from './DestinationPanel.tsx'
import { type ShownInfo, sharedMods } from './logic.ts'
import { shareDestinations } from './methods.ts'
import { CancelFooter, MessageFooter } from './ShareFooter.tsx'
import { useShareBuild } from './useShareBuild.ts'

function shareMessage(info: ShownInfo, gameName: string): string {
  const count = sharedMods(info)
  return plural(count, {
    one: `Try my Mortar profile "${info.name}" for ${gameName} (# mod): ${info.web}`,
    other: `Try my Mortar profile "${info.name}" for ${gameName} (# mods): ${info.web}`,
  })
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
  const entries = shareDestinations({ thunderstore, count: shareable })
  const names = {
    mortar: t`Mortar`,
    nexus: t`Nexus Mods`,
    thunderstore: t`Thunderstore`,
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
      maxWidth={false}
      aria-label={info ? t`Share ${info.name}` : undefined}
      slotProps={{
        paper: {
          sx: {
            bgcolor: 'var(--mortar-panel-solid)',
            border: '1px solid var(--mortar-hairline-12)',
            width: 'min(880px, calc(100% - 48px))',
            height: destination ? 'min(600px, calc(100% - 48px))' : 'auto',
            maxHeight: 'calc(100% - 48px)',
            overflow: 'hidden',
            borderRadius: '12px',
          },
        },
      }}
    >
      {info ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.25, p: '20px 24px 16px' }}>
            {destination ? (
              <TipIconButton label={t`Back`} onClick={() => setDestination(null)}>
                <ArrowLeft size={18} />
              </TipIconButton>
            ) : null}
            <Box sx={{ minWidth: 0 }}>
              <Typography noWrap={true} title={info.name} sx={{ fontSize: 20, fontWeight: 700 }}>
                {t`Share ${info.name}`}
              </Typography>
              <Typography noWrap={true} sx={{ color: 'text.secondary', fontSize: 14 }}>
                {destination
                  ? names[destination]
                  : plural(count, {
                      one: `${{ name: gameName }} · # mod`,
                      other: `${{ name: gameName }} · # mods`,
                    })}
              </Typography>
            </Box>
            <Box sx={{ flex: 1 }} />
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
          {destination === 'mortar' ? (
            <MessageFooter info={info} message={message} />
          ) : (
            destination === null && <CancelFooter onCancel={close} />
          )}
        </Box>
      ) : null}
    </Dialog>
  )
}
