import { Trans, useLingui } from '@lingui/react/macro'
import { ButtonBase, Menu, MenuItem } from '@mui/material'
import { Application } from '@wailsio/runtime'
import { useState } from 'react'
import { Logo } from '../brand/Logo.tsx'
import { openSettings } from '../settings/SettingsDialogStore.ts'
import { useToasts } from '../toasts/store.ts'

export function AppMenu() {
  const { t } = useLingui()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const close = () => setAnchor(null)
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
        aria-haspopup="menu"
        aria-expanded={anchor ? true : undefined}
        onClick={(e) => setAnchor(e.currentTarget)}
        sx={{
          '--wails-draggable': 'no-drag',
          gap: '10px',
          pl: '12px',
          pr: '14px',
          fontSize: 15,
          fontWeight: 600,
          fontFamily: 'inherit',
          color: 'inherit',
          bgcolor: 'rgba(0,0,0,0.3)',
        }}
      >
        <Logo size={20} />
        <Trans>Mortar</Trans>
      </ButtonBase>
      <Menu anchorEl={anchor} open={anchor !== null} onClose={close}>
        <MenuItem
          onClick={() => {
            close()
            openSettings()
          }}
        >
          <Trans>Settings</Trans>
        </MenuItem>
        <MenuItem
          onClick={() => {
            close()
            openSettings('about')
          }}
        >
          <Trans>About Mortar</Trans>
        </MenuItem>
        <MenuItem onClick={quit}>
          <Trans>Quit</Trans>
        </MenuItem>
      </Menu>
    </>
  )
}
