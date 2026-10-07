import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Button, ButtonGroup, ListItemIcon, ListItemText, Menu, MenuItem } from '@mui/material'
import { Check, ChevronDown, Copy, FileText, List, MessageSquare, Type } from 'lucide-react'
import { useState } from 'react'
import { useProfiles } from '../profiles/store.ts'
import { MenuAction } from '../shell/MenuAction.tsx'
import { copyText } from './copyText.ts'
import { formatModList, listItems, type ModListFormat } from './modList.ts'
import { useShareDialog } from './store.ts'

export function ListFormat({ disabled }: { disabled: boolean }) {
  const { t } = useLingui()
  const profileId = useShareDialog((s) => s.profileId)
  const keys = useShareDialog((s) => s.keys)
  const format = useShareDialog((s) => s.listFormat)
  const setFormat = useShareDialog((s) => s.setListFormat)
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === profileId))
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [partsAnchor, setPartsAnchor] = useState<HTMLElement | null>(null)
  const items = listItems(profile, keys)
  const labels = { enabled: t`Enabled`, disabled: t`Disabled` }
  const parts = formatModList(format, items, labels)
  const options: { id: ModListFormat; label: string; icon: typeof FileText }[] = [
    { id: 'markdown', label: t`Markdown`, icon: FileText },
    { id: 'plain', label: t`Plain text`, icon: Type },
    { id: 'discord', label: t`Discord`, icon: MessageSquare },
  ]
  return (
    <>
      <ButtonGroup variant="outlined" color="inherit">
        <Button
          startIcon={<List size={16} />}
          aria-haspopup={parts.length > 1 ? 'menu' : undefined}
          aria-expanded={parts.length > 1 ? partsAnchor !== null : undefined}
          onClick={(e) => {
            if (parts.length > 1) {
              setPartsAnchor(e.currentTarget)
              return
            }
            copyText(parts[0]?.text ?? '', t`Mod list copied`)
          }}
          disabled={disabled}
          sx={{ height: 40, px: '14px', fontSize: 14 }}
        >
          {parts.length > 1
            ? plural(parts.length, {
                one: 'Copy mod list (# part)',
                other: 'Copy mod list (# parts)',
              })
            : t`Copy mod list`}
        </Button>
        <Button
          aria-label={t`Mod list format`}
          endIcon={<ChevronDown size={14} />}
          aria-haspopup="menu"
          aria-expanded={anchor !== null}
          onClick={(e) => setAnchor(e.currentTarget)}
          sx={{ height: 40, px: '14px', fontSize: 14 }}
        >
          {options.find((o) => o.id === format)?.label}
        </Button>
      </ButtonGroup>
      <Menu open={Boolean(anchor)} anchorEl={anchor} onClose={() => setAnchor(null)}>
        {options.map((o) => {
          const Icon = o.icon
          return (
            <MenuItem
              key={o.id}
              role="menuitemradio"
              aria-checked={o.id === format}
              selected={o.id === format}
              onClick={() => {
                setFormat(o.id)
                setAnchor(null)
              }}
            >
              <ListItemIcon sx={{ color: 'inherit' }}>
                {o.id === format ? <Check size={16} /> : <Icon size={16} />}
              </ListItemIcon>
              <ListItemText>{o.label}</ListItemText>
            </MenuItem>
          )
        })}
      </Menu>
      <Menu open={Boolean(partsAnchor)} anchorEl={partsAnchor} onClose={() => setPartsAnchor(null)}>
        {parts.map((part) => (
          <MenuAction
            key={part.id}
            icon={<Copy size={16} />}
            label={t`Copy part ${part.n} of ${parts.length}`}
            onClick={() => {
              copyText(part.text, t`Part ${part.n} copied`)
              setPartsAnchor(null)
            }}
          />
        ))}
      </Menu>
    </>
  )
}
