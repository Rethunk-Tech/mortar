import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Dialog, Typography } from '@mui/material'
import { FileJson, List, MessageSquare, Package, Save, Send, Share2, X } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import { useProfiles } from '../profiles/store.ts'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { TipBanner } from '../tips/TipBanner.tsx'
import { copyText } from './copyText.ts'
import { MethodPanel } from './MethodPanel.tsx'
import { type ShareMethod, shareMethods } from './methods.ts'
import { PreviewDialog } from './PreviewDialog.tsx'
import { useShareBuild } from './useShareBuild.ts'

export function ShareDialog() {
  const { t } = useLingui()
  const gameName = useProfiles((s) => s.game?.name ?? '')
  const built = useShareBuild()
  const { profileId, close, method, setMethod, thunderstore, info } = built
  const [previewing, setPreviewing] = useState(false)
  const rail: Record<ShareMethod, { icon: ReactNode; name: string; hint: string }> = {
    link: {
      icon: <Share2 size={18} />,
      name: t`Link`,
      hint: t`Paste anywhere; they open it in Mortar`,
    },
    file: {
      icon: <Save size={18} />,
      name: t`File`,
      hint: t`.mortar, with your mod settings`,
    },
    list: {
      icon: <List size={18} />,
      name: t`Mod list`,
      hint: t`Text for Discord or a forum`,
    },
    nexus: {
      icon: <FileJson size={18} />,
      name: t`Nexus collection`,
      hint: t`A draft on Nexus Mods`,
    },
    nearby: {
      icon: <Send size={18} />,
      name: t`Nearby computer`,
      hint: t`Send over your network`,
    },
    thunderstore: {
      icon: <Package size={18} />,
      name: t`r2modman`,
      hint: t`A code or a Thunderstore modpack`,
    },
  }
  const count = info?.count ?? 0
  const message = info
    ? plural(count, {
        one: `Try my Mortar profile "${info.name}" for ${gameName} (# mod): ${info.web}`,
        other: `Try my Mortar profile "${info.name}" for ${gameName} (# mods): ${info.web}`,
      })
    : ''
  const methods = shareMethods({ thunderstore, count: info?.count ?? 0 })
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
            <Box
              role="tablist"
              aria-orientation="vertical"
              aria-label={t`How to share`}
              sx={{
                display: 'flex',
                flexDirection: 'column',
                gap: 0.5,
                p: '16px 12px',
                overflowY: 'auto',
                borderRight: '1px solid var(--mortar-hairline-muted)',
              }}
            >
              {methods.map((m) => {
                const selected = m.id === method
                const entry = rail[m.id]
                return (
                  <Button
                    key={m.id}
                    role="tab"
                    aria-selected={selected}
                    color="inherit"
                    disabled={m.disabled}
                    onClick={() => setMethod(m.id)}
                    sx={{
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'flex-start',
                      gap: 1.5,
                      p: '10px 12px',
                      textAlign: 'left',
                      textTransform: 'none',
                      borderRadius: '8px',
                      border: '1px solid',
                      borderColor: selected ? 'primary.main' : 'transparent',
                      bgcolor: selected ? 'var(--mortar-hairline-14)' : 'transparent',
                    }}
                  >
                    {entry.icon}
                    <Box sx={{ display: 'flex', flexDirection: 'column', minWidth: 0 }}>
                      <Box component="span" sx={{ fontWeight: 600, fontSize: 14 }}>
                        {entry.name}
                      </Box>
                      <Box component="span" sx={{ fontSize: 12, color: 'text.secondary' }}>
                        {entry.hint}
                      </Box>
                    </Box>
                  </Button>
                )
              })}
            </Box>
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
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: 1.25,
              p: '14px 24px',
              borderTop: '1px solid var(--mortar-hairline-muted)',
            }}
          >
            <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
              {thunderstore ? t`Thunderstore games also offer an r2modman code.` : ''}
            </Typography>
            <Box sx={{ flex: 1 }} />
            <Button
              variant="outlined"
              color="inherit"
              startIcon={<MessageSquare size={16} />}
              disabled={info.count === 0 || info.tooLarge}
              onClick={() => copyText(message, t`Message copied`)}
              sx={{ height: 34 }}
            >
              {t`Copy as a message`}
            </Button>
          </Box>
          <PreviewDialog info={info} open={previewing} onClose={() => setPreviewing(false)} />
        </Box>
      ) : null}
    </Dialog>
  )
}
