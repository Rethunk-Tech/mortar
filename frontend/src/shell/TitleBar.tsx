import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase } from '@mui/material'
import { Copy, Minus, Square, X } from 'lucide-react'
import type { ReactNode } from 'react'
import { GameMenu } from '../games/GameMenu.tsx'
import { routeGame, useNav } from '../nav/store.ts'
import { ProfileMenu } from '../profiles/ProfileMenu.tsx'
import { DownloadsPill } from '../queue/DownloadsPill.tsx'
import { HistoryButton } from '../toasts/HistoryButton.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { AppMenu } from './AppMenu.tsx'
import { win } from './win.ts'

const noDrag = { '--wails-draggable': 'no-drag' } as const

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
        width: 'var(--window-button)',
        color: 'var(--mortar-ink-85)',
        '&:hover': { bgcolor: danger ? 'error.main' : 'var(--mortar-hairline)' },
      }}
    >
      {children}
    </ButtonBase>
  )
}

const rule = { width: '1px', height: 22, flexShrink: 0, bgcolor: 'var(--mortar-hairline-14)' }

function Slash() {
  return (
    <Box
      component="span"
      aria-hidden={true}
      sx={{ color: 'var(--mortar-ink-dim-60)', '&::before': { content: '"/"' } }}
    />
  )
}

export function TitleBar({ maximised }: { maximised: boolean }) {
  const { t } = useLingui()
  const route = useNav((s) => s.route)
  const game = useNav((s) => routeGame(s.route))
  return (
    <Box
      component="header"
      onDoubleClick={(e) => {
        if (e.target instanceof Element && !e.target.closest('button, a')) {
          win.toggleMaximise().catch(reportUnexpected)
        }
      }}
      sx={{
        '--wails-draggable': 'drag',
        height: 'var(--title-bar)',
        flexShrink: 0,
        display: 'flex',
        alignItems: 'center',
        gap: '8px',
        pl: '14px',
        bgcolor: 'var(--mortar-title-bar)',
      }}
    >
      <AppMenu />
      <Box sx={rule} />
      {route.name === 'setup' ? (
        <Box component="span" sx={{ px: '10px', fontSize: 14, fontWeight: 600 }}>
          {t`Setup`}
        </Box>
      ) : (
        <GameMenu />
      )}
      {game ? (
        <>
          <Slash />
          <ProfileMenu game={game} />
        </>
      ) : null}
      {route.name === 'settings' ? (
        <>
          <Slash />
          <Box component="span" sx={{ fontSize: 14, fontWeight: 600 }}>
            {t`Settings`}
          </Box>
        </>
      ) : null}
      <Box sx={{ flexGrow: 1 }} />
      <DownloadsPill />
      <HistoryButton />
      <Box
        data-window-controls=""
        sx={{ display: 'flex', alignItems: 'stretch', alignSelf: 'stretch', ml: '6px' }}
      >
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
    </Box>
  )
}
