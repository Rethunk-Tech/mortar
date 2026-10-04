import { useLingui } from '@lingui/react/macro'
import { Box, Button, Dialog, Typography } from '@mui/material'
import { X } from 'lucide-react'
import type { ReactNode } from 'react'
import { paper } from '../mods/paper.ts'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { TipBanner } from '../tips/TipBanner.tsx'
import type { ShownInfo } from './logic.ts'
import { TabPills } from './TabPills.tsx'

export function ShareShell({
  art,
  onSendNearby,
  profileId,
  close,
  tab,
  setTab,
  info,
  headerExtra,
  children,
  preview,
}: {
  art?: string | undefined
  onSendNearby: (on: boolean) => void
  profileId: string
  close: () => void
  tab: 'link' | 'file'
  setTab: (tab: 'link' | 'file') => void
  info: ShownInfo | null
  headerExtra: ReactNode
  children: ReactNode
  preview: ReactNode
}) {
  const { t } = useLingui()
  return (
    <Dialog
      open={profileId !== '' && info !== null}
      onClose={close}
      maxWidth={false}
      transitionDuration={0}
      aria-label={info ? t`Share ${info.name}` : undefined}
      slotProps={{
        paper: {
          ...paper,
          sx: {
            ...paper.sx,
            bgcolor: 'var(--mortar-panel-solid)',
            border: '1px solid var(--mortar-hairline-12)',
            width: 'min(980px, calc(100% - 48px))',
            height: 'min(620px, calc(100% - 48px))',
            overflow: 'hidden',
            borderRadius: '10px',
          },
        },
      }}
    >
      {info ? (
        <Box
          sx={{
            position: 'relative',
            display: 'grid',
            gridTemplateColumns: tab === 'link' ? 'minmax(0, 1fr) 340px' : 'minmax(0, 1fr)',
            height: '100%',
            minHeight: 0,
          }}
        >
          <TipIconButton
            label={t`Close`}
            onClick={close}
            sx={{ position: 'absolute', top: 8, right: 8, zIndex: 1 }}
          >
            <X size={18} />
          </TipIconButton>
          <Box sx={{ display: 'flex', flexDirection: 'column', minWidth: 0, minHeight: 0 }}>
            <TipBanner tip="share">
              {t`A share link names this profile and the Nexus or GitHub files in it, not the archives.`}
            </TipBanner>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.75, p: '20px 24px 0' }}>
              {art ? (
                <Box
                  component="img"
                  src={art}
                  alt=""
                  sx={{ width: 64, height: 64, objectFit: 'cover', borderRadius: '8px' }}
                />
              ) : null}
              <Box sx={{ flex: 1, minWidth: 0 }}>
                <Typography
                  sx={{ fontSize: 13, color: 'text.secondary' }}
                >{t`Share profile`}</Typography>
                <Typography noWrap={true} title={info.name} sx={{ fontSize: 24, fontWeight: 700 }}>
                  {info.name}
                </Typography>
              </Box>
              {headerExtra}
            </Box>
            <Box sx={{ m: '18px 24px 0', display: 'flex', alignItems: 'center', gap: 1.5 }}>
              <TabPills
                value={tab}
                onChange={setTab}
                label={t`How to share`}
                options={[
                  { value: 'link', label: t`Link` },
                  { value: 'file', label: t`.mortar file with settings` },
                ]}
              />
              <Button
                variant="outlined"
                color="inherit"
                onClick={() => onSendNearby(true)}
                sx={{ height: 40, px: '14px', fontSize: 14, whiteSpace: 'nowrap' }}
              >
                {t`Send nearby…`}
              </Button>
            </Box>
            <Box sx={{ overflowY: 'auto', flex: 1, minHeight: 0 }}>{children}</Box>
          </Box>
          {preview}
        </Box>
      ) : null}
    </Dialog>
  )
}
