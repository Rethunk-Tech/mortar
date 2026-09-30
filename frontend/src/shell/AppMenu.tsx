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
import { Code2, FolderOpen, Info, LogOut, Settings } from 'lucide-react'
import { useId, useState } from 'react'
import { OpenDataFolder } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { Logo } from '../brand/Logo.tsx'
import { compact } from '../game/compact.ts'
import { openSettings } from '../nav/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { NexusAccount } from './NexusAccount.tsx'

const SOURCE = 'https://github.com/Rethunk-AI/mortar'

export function AppMenu() {
  const { t } = useLingui()
  const drawerId = useId()
  const [open, setOpen] = useState(false)
  const close = () => setOpen(false)
  const quit = () => {
    close()
    Application.Quit().catch((e: unknown) =>
      useToasts.getState().push({
        kind: 'error',
        title: t`Could not quit Mortar`,
        body: e instanceof Error ? e.message : String(e),
      }),
    )
  }
  return (
    <>
      <ButtonBase
        aria-label={t`Mortar menu`}
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
          bgcolor: 'rgba(0,0,0,0.3)',
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
              bgcolor: 'rgba(40,40,48,0.92)',
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
