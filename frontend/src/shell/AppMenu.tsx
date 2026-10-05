import { useLingui } from '@lingui/react/macro'
import {
  ButtonBase,
  Divider,
  Drawer,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
} from '@mui/material'
import { Application } from '@wailsio/runtime'
import {
  Bug,
  Code2,
  FolderOpen,
  Info,
  LifeBuoy,
  LogOut,
  RefreshCw,
  Settings,
  Sparkles,
} from 'lucide-react'
import { useEffect, useId, useState } from 'react'
import { OpenDataFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { Logo } from '../brand/Logo.tsx'
import { useConsole } from '../console/store.ts'
import { compact } from '../game/compact.ts'
import { useTab } from '../game/tab.ts'
import { openPage } from '../mods/menu.ts'
import { openSettings, routeGame, useNav } from '../nav/store.ts'
import { useMortarUpdate } from '../settings/updates.ts'
import { reportUnexpected, toastError } from '../toasts/report.ts'
import { showWhatsNew } from '../updates/whatsNew.ts'
import { checkForUpdates } from './checkForUpdates.ts'
import { NexusAccount } from './NexusAccount.tsx'
import { reportBug } from './reportBug.ts'

const SOURCE = 'https://github.com/Rethunk-Tech/mortar'

export function AppMenu() {
  const { t } = useLingui()
  const drawerId = useId()
  const [open, setOpen] = useState(false)
  const game = useNav((s) => routeGame(s.route) ?? '')
  const version = useMortarUpdate((s) => s.info?.version)
  useEffect(() => {
    useMortarUpdate
      .getState()
      .load()
      .catch(() => undefined)
  }, [])
  const close = () => setOpen(false)
  const quit = () => {
    close()
    Application.Quit().catch((e: unknown) => toastError(t`Could not quit Mortar`, e))
  }
  return (
    <>
      <ButtonBase
        aria-label={t`Mortar menu`}
        data-tour="app-menu"
        aria-expanded={open}
        aria-controls={open ? drawerId : undefined}
        onClick={() => setOpen(true)}
        sx={{
          '--wails-draggable': 'no-drag',
          gap: '10px',
          pl: '12px',
          pr: '14px',
          fontSize: 15,
          fontWeight: 600,
          [compact]: { pl: '11px', pr: '11px', '& .label': { display: 'none' } },
          fontFamily: 'inherit',
          color: 'inherit',
          bgcolor: 'var(--mortar-overlay-30)',
        }}
      >
        <Logo size={20} />
        <span className="label">{t`Mortar`}</span>
      </ButtonBase>
      <Drawer
        id={drawerId}
        anchor="left"
        open={open}
        onClose={close}
        sx={{ top: 'var(--title-bar)' }}
        slotProps={{
          paper: {
            role: 'dialog',
            'aria-label': t`Mortar menu`,
            sx: {
              width: 280,
              top: 'var(--title-bar)',
              height: 'calc(100% - var(--title-bar))',
              bgcolor: 'var(--mortar-panel-92)',
            },
          },
        }}
      >
        <List component="nav" aria-label={t`Mortar menu`}>
          <ListItemButton
            onClick={() => {
              close()
              openSettings()
            }}
          >
            <ListItemIcon>
              <Settings size={18} />
            </ListItemIcon>
            <ListItemText primary={t`Settings`} />
          </ListItemButton>
          <ListItemButton
            onClick={() => {
              close()
              checkForUpdates().catch(reportUnexpected)
            }}
          >
            <ListItemIcon>
              <RefreshCw size={18} />
            </ListItemIcon>
            <ListItemText primary={t`Check for updates`} />
          </ListItemButton>
          {version ? (
            <ListItemButton
              onClick={() => {
                close()
                showWhatsNew(version)
              }}
            >
              <ListItemIcon>
                <Sparkles size={18} />
              </ListItemIcon>
              <ListItemText primary={t`What's new in ${version}`} />
            </ListItemButton>
          ) : null}
          <Divider />
          <ListItemButton
            onClick={() => {
              close()
              OpenDataFolder().catch(reportUnexpected)
            }}
          >
            <ListItemIcon>
              <FolderOpen size={18} />
            </ListItemIcon>
            <ListItemText primary={t`Open data folder`} />
          </ListItemButton>
          <ListItemButton
            disabled={game === ''}
            onClick={() => {
              close()
              useTab.getState().setTab('console')
              useConsole.getState().setHelping(true)
            }}
          >
            <ListItemIcon>
              <LifeBuoy size={18} />
            </ListItemIcon>
            <ListItemText primary={t`Get help`} />
          </ListItemButton>
          <ListItemButton
            onClick={() => {
              close()
              reportBug(game)
            }}
          >
            <ListItemIcon>
              <Bug size={18} />
            </ListItemIcon>
            <ListItemText primary={t`Report a bug`} />
          </ListItemButton>
          <Divider />
          <ListItemButton
            onClick={() => {
              close()
              openSettings('about')
            }}
          >
            <ListItemIcon>
              <Info size={18} />
            </ListItemIcon>
            <ListItemText primary={t`About Mortar`} />
          </ListItemButton>
          <ListItemButton
            onClick={() => {
              close()
              openPage(SOURCE).catch(reportUnexpected)
            }}
          >
            <ListItemIcon>
              <Code2 size={18} />
            </ListItemIcon>
            <ListItemText primary={t`Source code`} />
          </ListItemButton>
          <Divider />
          <ListItemButton onClick={quit}>
            <ListItemIcon>
              <LogOut size={18} />
            </ListItemIcon>
            <ListItemText primary={t`Quit`} />
          </ListItemButton>
        </List>
        <NexusAccount onNavigate={close} />
      </Drawer>
    </>
  )
}
