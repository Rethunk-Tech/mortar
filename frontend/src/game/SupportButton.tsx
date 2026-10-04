import { useLingui } from '@lingui/react/macro'
import { IconButton, Menu, Tooltip } from '@mui/material'
import { Bug, LifeBuoy } from 'lucide-react'
import { useState } from 'react'
import { useConsole } from '../console/store.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { reportBug } from '../shell/reportBug.ts'
import { useTab } from './tab.ts'

export function SupportButton({ game }: { game: string }) {
  const { t } = useLingui()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const close = () => setAnchor(null)
  return (
    <>
      <Tooltip title={t`Support`}>
        <IconButton
          aria-label={t`Support`}
          aria-haspopup="menu"
          onClick={(e) => setAnchor(e.currentTarget)}
          sx={{ width: 40, height: 40, borderRadius: '6px' }}
        >
          <LifeBuoy size={18} />
        </IconButton>
      </Tooltip>
      <Menu open={anchor !== null} anchorEl={anchor} onClose={close}>
        <MenuAction
          icon={<LifeBuoy size={16} />}
          label={t`Get help`}
          onClick={() => {
            close()
            useTab.getState().setTab('console')
            useConsole.getState().setHelping(true)
          }}
        />
        <MenuAction
          icon={<Bug size={16} />}
          label={t`Report a Mortar bug`}
          onClick={() => {
            close()
            reportBug(game)
          }}
        />
      </Menu>
    </>
  )
}
