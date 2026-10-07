import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Dialog, IconButton, Typography } from '@mui/material'
import { X } from 'lucide-react'
import { Logo } from '../brand/Logo.tsx'
import { heading } from '../mods/paper.ts'
import { useProfiles } from '../profiles/store.ts'
import type { ShownInfo } from './logic.ts'

const PAGE_HOST = 'mortar.rethunk.tech'

function PagePreview({ info, onClose }: { info: ShownInfo; onClose: () => void }) {
  const { t } = useLingui()
  const art = useProfiles((s) => s.game?.artUrl)
  const gameName = useProfiles((s) => s.game?.name ?? '')
  const { count } = info
  const button = {
    display: 'grid',
    placeItems: 'center',
    height: 40,
    borderRadius: '6px',
    fontSize: 14,
    whiteSpace: 'nowrap',
  } as const
  return (
    <Box
      component="aside"
      aria-label={t`What the person opening the link sees`}
      sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, p: 2.5 }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Typography sx={{ ...heading, display: 'flex', alignItems: 'center' }}>
          {t`What they see`}
        </Typography>
        <IconButton size="small" aria-label={t`Close`} onClick={onClose}>
          <X size={18} />
        </IconButton>
      </Box>
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: 1.5,
          p: '18px',
          bgcolor: '#f4f1ea',
          color: '#1b1a17',
          borderRadius: '8px',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, fontSize: 13, fontWeight: 700 }}>
          <Logo size={16} fill="#1b1a17" />
          {PAGE_HOST}
        </Box>
        {art ? (
          <Box
            component="img"
            src={art}
            alt=""
            sx={{ width: '100%', height: 96, objectFit: 'cover', borderRadius: '6px' }}
          />
        ) : null}
        <Typography
          noWrap={true}
          title={info.name}
          sx={{ fontSize: 20, fontWeight: 700, color: 'inherit' }}
        >
          {info.name}
        </Typography>
        <Typography sx={{ fontSize: 13, color: 'inherit' }}>
          {plural(count, {
            one: `${{ name: gameName }} · # mod`,
            other: `${{ name: gameName }} · # mods`,
          })}
        </Typography>
        <Box sx={{ ...button, bgcolor: '#1b1a17', color: '#f4f1ea', fontWeight: 700 }}>
          {t`Open in Mortar`}
        </Box>
        <Box sx={{ ...button, border: '1px solid #1b1a17' }}>{t`Get Mortar`}</Box>
      </Box>
      <Typography sx={{ fontSize: 12, color: 'text.secondary', lineHeight: 1.5 }}>
        {t`The page reads the profile from the link itself. Nothing is uploaded or stored on a server.`}
      </Typography>
    </Box>
  )
}

export function PreviewDialog({
  info,
  open,
  onClose,
}: {
  info: ShownInfo
  open: boolean
  onClose: () => void
}) {
  const { t } = useLingui()
  return (
    <Dialog
      open={open}
      onClose={onClose}
      slotProps={{
        paper: {
          'aria-label': t`What they see`,
          sx: {
            bgcolor: 'var(--mortar-panel-solid)',
            border: '1px solid var(--mortar-hairline-12)',
            width: 380,
          },
        },
      }}
    >
      <PagePreview info={info} onClose={onClose} />
    </Dialog>
  )
}
