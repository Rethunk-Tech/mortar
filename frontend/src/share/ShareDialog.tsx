import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Dialog, Typography } from '@mui/material'
import { X } from 'lucide-react'
import { useState } from 'react'
import { useProfiles } from '../profiles/store.ts'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { TipBanner } from '../tips/TipBanner.tsx'
import { MethodPanel } from './MethodPanel.tsx'
import { MethodRail } from './MethodRail.tsx'
import { shareMethods } from './methods.ts'
import { PreviewDialog } from './PreviewDialog.tsx'
import { ShareFooter } from './ShareFooter.tsx'
import { useShareBuild } from './useShareBuild.ts'

export function ShareDialog() {
  const { t } = useLingui()
  const gameName = useProfiles((s) => s.game?.name ?? '')
  const built = useShareBuild()
  const { profileId, close, method, setMethod, thunderstore, info } = built
  const [previewing, setPreviewing] = useState(false)
  const count = info?.count ?? 0
  const message = info
    ? plural(count, {
        one: `Try my Mortar profile "${info.name}" for ${gameName} (# mod): ${info.web}`,
        other: `Try my Mortar profile "${info.name}" for ${gameName} (# mods): ${info.web}`,
      })
    : ''
  const methods = shareMethods({ thunderstore, count })
  return (
    <Dialog
      open={profileId !== '' && info !== null}
      onClose={close}
      maxWidth={false}
      aria-label={info ? t`Share ${info.name}` : undefined}
      slotProps={{
        paper: {
          sx: {
            bgcolor: 'var(--mortar-panel-solid)',
            border: '1px solid var(--mortar-hairline-12)',
            width: 'min(880px, calc(100% - 48px))',
            height: 'min(600px, calc(100% - 48px))',
            overflow: 'hidden',
            borderRadius: '12px',
          },
        },
      }}
    >
      {info ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', minHeight: 0 }}>
          <Box sx={{ display: 'flex', alignItems: 'baseline', gap: 1.25, p: '20px 24px 16px' }}>
            <Typography noWrap={true} title={info.name} sx={{ fontSize: 20, fontWeight: 700 }}>
              {t`Share ${info.name}`}
            </Typography>
            <Typography noWrap={true} sx={{ color: 'text.secondary' }}>
              {plural(count, {
                one: `${{ name: gameName }} · # mod`,
                other: `${{ name: gameName }} · # mods`,
              })}
            </Typography>
            <Box sx={{ flex: 1 }} />
            <TipIconButton label={t`Close`} onClick={close}>
              <X size={18} />
            </TipIconButton>
          </Box>
          <TipBanner tip="share">
            {t`A share link names this profile and where each mod comes from, not the files themselves.`}
          </TipBanner>
          <Box
            sx={{
              flex: 1,
              display: 'grid',
              gridTemplateColumns: '250px minmax(0, 1fr)',
              minHeight: 0,
              borderTop: '1px solid var(--mortar-hairline-muted)',
            }}
          >
            <MethodRail method={method} setMethod={setMethod} methods={methods} />
            <Box
              role="tabpanel"
              sx={{
                display: 'flex',
                flexDirection: 'column',
                p: '24px 28px',
                minWidth: 0,
                overflowY: 'auto',
              }}
            >
              <MethodPanel built={built} />
              <Box sx={{ flex: 1 }} />
              <Box>
                <Button
                  variant="text"
                  disabled={info.tooLarge}
                  onClick={() => setPreviewing(true)}
                  sx={{ p: 0, minWidth: 0 }}
                >
                  {t`Preview what they see`}
                </Button>
              </Box>
            </Box>
          </Box>
          <ShareFooter thunderstore={thunderstore} info={info} message={message} />
          <PreviewDialog info={info} open={previewing} onClose={() => setPreviewing(false)} />
        </Box>
      ) : null}
    </Dialog>
  )
}
