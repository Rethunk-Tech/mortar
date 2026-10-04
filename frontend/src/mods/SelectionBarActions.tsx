import { useLingui } from '@lingui/react/macro'
import { Box, Button, ListItemIcon, ListItemText, Menu, MenuItem, Typography } from '@mui/material'
import {
  BellOff,
  BellRing,
  Copy,
  Ellipsis,
  Folder,
  PackagePlus,
  Pin,
  PinOff,
  Power,
  PowerOff,
  Share2,
  Tag,
  Trash2,
  X,
} from 'lucide-react'
import { type ReactNode, useState } from 'react'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { IconAction } from '../shell/IconAction.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { LockedReason } from './LockedReason.tsx'
import type { SourceGroups } from './skipSources.ts'

export function SelectionBarActions({
  selected,
  profile,
  latest,
  locked,
  count,
  setEnabledMany,
  setPinnedMany,
  setSkipVersionMany,
  sourceGroups,
  setSkipSourceMany,
  askRemove,
  openAlso,
  openSave,
  openTag,
  openCategory,
  share,
  clear,
}: {
  selected: Mod[]
  profile: { entries?: { key: string; pinned?: boolean }[] | null } | undefined
  latest: ReadonlyMap<string, string>
  locked: boolean
  count: string
  setEnabledMany: (mods: Mod[], enabled: boolean) => Promise<void>
  setPinnedMany: (mods: Mod[], pinned: boolean) => Promise<void>
  setSkipVersionMany: (mods: Mod[]) => Promise<void>
  sourceGroups: SourceGroups
  setSkipSourceMany: (groups: ReadonlyMap<string, Mod[]>, skip: boolean) => Promise<void>
  askRemove: (mods: Mod[]) => void
  openAlso: () => void
  openSave: () => void
  openTag: () => void
  openCategory: () => void
  share: () => void
  clear: () => void
}) {
  const { t } = useLingui()
  const [menu, setMenu] = useState<HTMLElement | null>(null)
  const pinned = selected.every(
    (mod) => profile?.entries?.find((entry) => entry.key === mod.key)?.pinned,
  )
  const pick = (run: () => void) => () => {
    setMenu(null)
    run()
  }
  const more: { label: string; icon: ReactNode; run: () => void; edits: boolean }[] = [
    { label: t`Tag…`, icon: <Tag size={16} />, run: openTag, edits: true },
    { label: t`Set category…`, icon: <Folder size={16} />, run: openCategory, edits: true },
    {
      label: pinned ? t`Unpin version` : t`Keep this version`,
      icon: pinned ? <PinOff size={16} /> : <Pin size={16} />,
      run: () => setPinnedMany(selected, !pinned).catch(reportUnexpected),
      edits: true,
    },
    {
      label: t`Skip this update`,
      icon: <BellOff size={16} />,
      run: () =>
        setSkipVersionMany(selected.filter((mod) => latest.has(mod.key))).catch(reportUnexpected),
      edits: true,
    },
    ...(sourceGroups.ignore.size > 0
      ? [
          {
            label: t`Ignore update source`,
            icon: <BellOff size={16} />,
            run: () => setSkipSourceMany(sourceGroups.ignore, true).catch(reportUnexpected),
            edits: true,
          },
        ]
      : []),
    ...(sourceGroups.use.size > 0
      ? [
          {
            label: t`Use update source`,
            icon: <BellRing size={16} />,
            run: () => setSkipSourceMany(sourceGroups.use, false).catch(reportUnexpected),
            edits: true,
          },
        ]
      : []),
    { label: t`Also add to…`, icon: <Copy size={16} />, run: openAlso, edits: true },
    { label: t`Save as bundle…`, icon: <PackagePlus size={16} />, run: openSave, edits: true },
    { label: t`Share selection`, icon: <Share2 size={16} />, run: share, edits: false },
  ]
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        px: 2,
        py: 0.75,
        minHeight: 44,
        borderBottom: '1px solid var(--mortar-hairline-muted)',
        overflow: 'hidden',
      }}
    >
      <Typography sx={{ fontSize: 14, fontWeight: 600, mr: 'auto', whiteSpace: 'nowrap' }}>
        {count}
      </Typography>
      <LockedReason locked={locked}>
        <Box sx={{ display: 'inline-flex', alignItems: 'center', gap: 1 }}>
          <Button
            size="small"
            variant="outlined"
            disabled={locked}
            startIcon={<Power size={15} />}
            onClick={() => setEnabledMany(selected, true).catch(reportUnexpected)}
          >{t`Switch on`}</Button>
          <Button
            size="small"
            variant="outlined"
            disabled={locked}
            startIcon={<PowerOff size={15} />}
            onClick={() => setEnabledMany(selected, false).catch(reportUnexpected)}
          >{t`Switch off`}</Button>
          <Button
            size="small"
            variant="outlined"
            color="error"
            disabled={locked}
            startIcon={<Trash2 size={15} />}
            onClick={() => askRemove(selected)}
          >{t`Remove`}</Button>
        </Box>
      </LockedReason>
      <IconAction
        label={t`More actions`}
        icon={<Ellipsis size={18} />}
        menu={true}
        aria-haspopup="menu"
        aria-expanded={menu !== null}
        onClick={(e) => setMenu(e.currentTarget)}
      />
      <IconAction label={t`Clear selection`} icon={<X size={18} />} onClick={clear} />
      <Menu anchorEl={menu} open={menu !== null} onClose={() => setMenu(null)}>
        {more.map((m) => (
          <MenuItem key={m.label} disabled={m.edits && locked} onClick={pick(m.run)}>
            <ListItemIcon>{m.icon}</ListItemIcon>
            <ListItemText>{m.label}</ListItemText>
          </MenuItem>
        ))}
      </Menu>
    </Box>
  )
}
