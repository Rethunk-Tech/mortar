import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase } from '@mui/material'
import { Copy, Minus, Square, X } from 'lucide-react'
import { memo, type ReactNode } from 'react'
import { GameMenu } from '../games/GameMenu.tsx'
import { routeGame, useNav } from '../nav/store.ts'
import { ProfileMenu } from '../profiles/ProfileMenu.tsx'
import { DownloadsPill } from '../queue/DownloadsPill.tsx'
import { useSettings } from '../settings/store.ts'
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

const Slash = memo(function SlashView() {
  return (
    <Box
      component="span"
      aria-hidden={true}
      sx={{ color: 'var(--mortar-ink-dim-60)', '&::before': { content: '"/"' } }}
    />
  )
})

// Memoised apart from the bar, which repaints on every route change while the buttons stay as they were.
const WindowControls = memo(function WindowControlsView({ maximised }: { maximised: boolean }) {
  const { t } = useLingui()
  return (
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
  )
})

export function TitleBar({ maximised }: { maximised: boolean }) {
  const { t } = useLingui()
  const routeName = useNav((s) => s.route.name)
  const game = useNav((s) => routeGame(s.route))
  // With the desktop's own title bar above it, this row is a toolbar: the desktop moves and sizes the window.
  const systemTitleBar = useSettings((s) => s.systemTitleBar)
  return (
    <Box
      component="header"
      onDoubleClick={(e) => {
        if (!systemTitleBar && e.target instanceof Element && !e.target.closest('button, a')) {
          win.toggleMaximise().catch(reportUnexpected)
        }
      }}
      sx={{
        '--wails-draggable': systemTitleBar ? 'no-drag' : 'drag',
        height: 'var(--title-bar)',
        flexShrink: 0,
        display: 'flex',
        alignItems: 'center',
        gap: '8px',
        pl: '14px',
        pr: systemTitleBar ? '8px' : 0,
        bgcolor: 'var(--mortar-title-bar)',
      }}
    >
      <AppMenu />
      <Box sx={rule} />
      {routeName === 'setup' ? (
        <Box component="span" sx={{ px: '10px', fontSize: 'medium.fontSize', fontWeight: 600 }}>
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
      {routeName === 'settings' ? (
        <>
          <Slash />
          <Box component="span" sx={{ fontSize: 'medium.fontSize', fontWeight: 600 }}>
            {t`Settings`}
          </Box>
        </>
      ) : null}
      <Box sx={{ flexGrow: 1 }} />
      <DownloadsPill />
      <HistoryButton />
      {systemTitleBar ? null : <WindowControls maximised={maximised} />}
    </Box>
  )
}
