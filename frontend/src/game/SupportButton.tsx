import { useLingui } from '@lingui/react/macro'
import { IconButton, ListItemIcon, ListItemText, Menu, MenuItem } from '@mui/material'
import { Bug, LifeBuoy } from 'lucide-react'
import { useState } from 'react'
import { useConsole } from '../console/store.ts'
import { reportBug } from '../shell/reportBug.ts'
import { useTab } from './tab.ts'

export function SupportButton({ game }: { game: string }) {
  const { t } = useLingui()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const close = () => setAnchor(null)
  return (
    <>
      <IconButton
        aria-label={t`Support`}
        aria-haspopup="menu"
        onClick={(e) => setAnchor(e.currentTarget)}
        sx={{ width: 40, height: 40, borderRadius: '6px' }}
      >
        <LifeBuoy size={18} />
      </IconButton>
      <Menu open={anchor !== null} anchorEl={anchor} onClose={close}>
        <MenuItem
          onClick={() => {
            close()
            useTab.getState().setTab('console')
            useConsole.getState().setHelping(true)
          }}
        >
          <ListItemIcon sx={{ color: 'inherit' }}>
            <LifeBuoy size={16} />
          </ListItemIcon>
          <ListItemText>{t`Get help`}</ListItemText>
        </MenuItem>
        <MenuItem
          onClick={() => {
            close()
            reportBug(game)
          }}
        >
          <ListItemIcon sx={{ color: 'inherit' }}>
            <Bug size={16} />
          </ListItemIcon>
          <ListItemText>{t`Report a Mortar bug`}</ListItemText>
        </MenuItem>
      </Menu>
    </>
  )
}
