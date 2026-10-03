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
import { Application, Browser } from '@wailsio/runtime'
import {
  Bug,
  Code2,
  FileArchive,
  FolderOpen,
  Info,
  LogOut,
  RefreshCw,
  Settings,
} from 'lucide-react'
import { useId, useState } from 'react'
import { OpenDataFolder } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { Logo } from '../brand/Logo.tsx'
import { compact } from '../game/compact.ts'
import { openSettings, routeGame, useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useMortarUpdate } from '../settings/updates.ts'
import { errorDetails, errorMessage, reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { NexusAccount } from './NexusAccount.tsx'
import { reportBug } from './reportBug.ts'
import { saveDiagnostics } from './saveDiagnostics.ts'

const SOURCE = 'https://github.com/Rethunk-AI/mortar'

export function AppMenu() {
  const { t } = useLingui()
  const drawerId = useId()
  const [open, setOpen] = useState(false)
  const game = useNav((s) => routeGame(s.route) ?? '')
  const profile = useProfiles((s) => s.openId)
  const close = () => setOpen(false)
  const quit = () => {
    close()
    Application.Quit().catch((e: unknown) =>
      useToasts.getState().push({
        kind: 'error',
        title: t`Could not quit Mortar`,
        body: errorMessage(e),
        detail: errorDetails(e),
      }),
    )
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
              openSettings('updates')
              useMortarUpdate.getState().check().catch(reportUnexpected)
            }}
          >
            <ListItemIcon>
              <RefreshCw size={18} />
            </ListItemIcon>
            <ListItemText primary={t`Check for updates`} />
          </ListItemButton>
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
            onClick={() => {
              close()
              saveDiagnostics(game, profile)
            }}
          >
            <ListItemIcon>
              <FileArchive size={18} />
            </ListItemIcon>
            <ListItemText primary={t`Save diagnostics…`} />
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
              Browser.OpenURL(SOURCE).catch(reportUnexpected)
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
