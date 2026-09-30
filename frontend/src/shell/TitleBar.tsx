import { Trans, useLingui } from '@lingui/react/macro'
import { Box, ButtonBase } from '@mui/material'
import { Copy, Minus, Square, X } from 'lucide-react'
import type { ReactNode } from 'react'
import { Logo } from '../brand/Logo.tsx'
import { useNav } from '../nav/store.ts'
import { win } from './win.ts'

const noDrag = { '--wails-draggable': 'no-drag' } as const

function Tab({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick?: () => void
  children: ReactNode
}) {
  return (
    <ButtonBase
      onClick={onClick}
      aria-current={active ? 'page' : undefined}
      sx={{
        ...noDrag,
        px: '12px',
        fontSize: 15,
        fontFamily: 'inherit',
        whiteSpace: 'nowrap',
        color: active ? '#ffffff' : 'rgba(210,210,215,0.92)',
        borderBottom: '2px solid',
        borderColor: active ? 'primary.main' : 'transparent',
      }}
    >
      {children}
    </ButtonBase>
  )
}

function WindowButton({
  label,
  onClick,
  danger,
  children,
}: {
  label: string
  onClick: () => void
  danger?: boolean
  children: ReactNode
}) {
  return (
    <ButtonBase
      aria-label={label}
      onClick={onClick}
      sx={{
        ...noDrag,
        width: 46,
        color: 'rgba(255,255,255,0.85)',
        '&:hover': { bgcolor: danger ? 'error.main' : 'rgba(255,255,255,0.1)' },
      }}
    >
      {children}
    </ButtonBase>
  )
}

export function TitleBar({ maximised }: { maximised: boolean }) {
  const { t } = useLingui()
  const route = useNav((s) => s.route)
  const openGame = useNav((s) => s.openGame)
  const openGameSelect = useNav((s) => s.openGameSelect)
  return (
    <Box
      component="header"
      onDoubleClick={win.toggleMaximise}
      sx={{
        '--wails-draggable': 'drag',
        height: 36,
        flexShrink: 0,
        display: 'flex',
        alignItems: 'stretch',
        bgcolor: 'rgba(15,15,18,0.55)',
      }}
    >
      <Box sx={{ width: 44, display: 'grid', placeItems: 'center', bgcolor: 'rgba(0,0,0,0.3)' }}>
        <Logo size={20} />
      </Box>
      <Box
        sx={{ display: 'flex', alignItems: 'center', px: '14px', fontSize: 15, fontWeight: 600 }}
      >
        <Trans>Mortar</Trans>
      </Box>
      <Tab active={route.name === 'game-select'} onClick={openGameSelect}>
        <Trans>Game Select</Trans>
      </Tab>
      {route.name === 'game' && (
        <Tab active={true} onClick={() => openGame(route.game)}>
          <Trans>Stardew Valley</Trans>
        </Tab>
      )}
      <Box sx={{ flexGrow: 1 }} />
      <WindowButton label={t`Minimise`} onClick={win.minimise}>
        <Minus size={14} />
      </WindowButton>
      <WindowButton label={maximised ? t`Restore` : t`Maximise`} onClick={win.toggleMaximise}>
        {maximised ? <Copy size={12} /> : <Square size={12} />}
      </WindowButton>
      <WindowButton label={t`Close`} onClick={win.close} danger={true}>
        <X size={14} />
      </WindowButton>
    </Box>
  )
}
