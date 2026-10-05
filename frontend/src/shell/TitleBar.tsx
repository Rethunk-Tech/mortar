import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase } from '@mui/material'
import { Copy, Minus, Square, X } from 'lucide-react'
import { type ReactNode, useEffect, useState } from 'react'
import { compact } from '../game/compact.ts'
import { GameSwitcher } from '../games/GameSwitcher.tsx'
import { useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { AppMenu } from './AppMenu.tsx'
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
        [compact]: { fontSize: 14 },
        fontFamily: 'inherit',
        whiteSpace: 'nowrap',
        color: active ? 'var(--mortar-ink)' : 'var(--mortar-ink-dim-92)',
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
        width: 'var(--window-button)',
        color: 'var(--mortar-ink-85)',
        '&:hover': { bgcolor: danger ? 'error.main' : 'var(--mortar-hairline)' },
      }}
    >
      {children}
    </ButtonBase>
  )
}

export function TitleBar({ maximised }: { maximised: boolean }) {
  const { t } = useLingui()
  const route = useNav((s) => s.route)
  const openGameSelect = useNav((s) => s.openGameSelect)
  const gameName = useProfiles((s) => s.game?.name)
  const [switching, setSwitching] = useState(false)
  // Any navigation, including choosing a game in the switcher, closes it.
  useEffect(() => useNav.subscribe(() => setSwitching(false)), [])
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
        alignItems: 'stretch',
        bgcolor: 'var(--mortar-title-bar)',
      }}
    >
      <AppMenu />
      {route.name === 'setup' ? (
        <Tab active={true} onClick={() => undefined}>
          {t`Setup`}
        </Tab>
      ) : (
        <Tab active={route.name === 'game-select'} onClick={openGameSelect}>
          {t`Game select`}
        </Tab>
      )}
      {(route.name === 'game' || route.name === 'profiles' || route.name === 'game-settings') && (
        <>
          <Tab active={true} onClick={() => setSwitching((on) => !on)}>
            {gameName ?? t`Game`}
          </Tab>
          <GameSwitcher current={route.game} open={switching} onClose={() => setSwitching(false)} />
        </>
      )}
      {route.name === 'settings' && (
        <Tab active={true} onClick={() => undefined}>
          {t`Settings`}
        </Tab>
      )}
      <Box sx={{ flexGrow: 1 }} />
      <Box data-window-controls="" sx={{ display: 'flex', alignItems: 'stretch' }}>
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
