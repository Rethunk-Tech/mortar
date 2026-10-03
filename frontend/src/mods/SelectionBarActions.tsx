import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { PackagePlus, Pin, PinOff, Power, PowerOff, Share2, Tag, Trash2, X } from 'lucide-react'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { reportUnexpected } from '../toasts/report.ts'

const noWrap = { whiteSpace: 'nowrap' } as const

export function SelectionBarActions({
  selected,
  profile,
  latest,
  locked,
  count,
  setEnabledMany,
  setPinnedMany,
  setSkipVersionMany,
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
  askRemove: (mods: Mod[]) => void
  openAlso: () => void
  openSave: () => void
  openTag: () => void
  openCategory: () => void
  share: () => void
  clear: () => void
}) {
  const { t } = useLingui()
  const pinned = selected.every(
    (mod) => profile?.entries?.find((entry) => entry.key === mod.key)?.pinned,
  )
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        px: 2,
        py: 0.75,
        minHeight: 40,
        borderBottom: '1px solid rgba(255,255,255,0.08)',
        flexWrap: 'wrap',
      }}
    >
      <Typography sx={{ fontSize: 13, mr: 0.5, whiteSpace: 'nowrap' }}>{count}</Typography>
      <DisabledReason title={t`Stop the game to change mods.`} disabled={locked}>
        <Box sx={{ display: 'inline-flex', flexWrap: 'wrap', alignItems: 'center', gap: 1 }}>
          <Button
            size="small"
            variant="outlined"
            disabled={locked}
            startIcon={<Power size={15} />}
            onClick={() => setEnabledMany(selected, true).catch(reportUnexpected)}
            sx={noWrap}
          >{t`Enable`}</Button>
          <Button
            size="small"
            variant="outlined"
            disabled={locked}
            startIcon={<PowerOff size={15} />}
            onClick={() => setEnabledMany(selected, false).catch(reportUnexpected)}
            sx={noWrap}
          >{t`Disable`}</Button>
          <Button
            size="small"
            variant="outlined"
            disabled={locked}
            startIcon={<Tag size={15} />}
            onClick={openTag}
            sx={noWrap}
          >{t`Tag`}</Button>
          <Button
            size="small"
            variant="outlined"
            disabled={locked}
            onClick={openCategory}
            sx={noWrap}
          >{t`Set category`}</Button>
          <Button
            size="small"
            variant="outlined"
            disabled={locked}
            startIcon={pinned ? <PinOff size={15} /> : <Pin size={15} />}
            onClick={() => setPinnedMany(selected, !pinned).catch(reportUnexpected)}
            sx={noWrap}
          >
            {pinned ? t`Unpin version` : t`Pin version`}
          </Button>
          <Button
            size="small"
            variant="outlined"
            disabled={locked}
            onClick={() =>
              setSkipVersionMany(selected.filter((mod) => latest.has(mod.key))).catch(
                reportUnexpected,
              )
            }
            sx={noWrap}
          >{t`Skip current updates`}</Button>
          <Button
            size="small"
            variant="outlined"
            disabled={locked}
            onClick={openAlso}
            sx={noWrap}
          >{t`Also add to…`}</Button>
          <Button
            size="small"
            variant="outlined"
            disabled={locked}
            startIcon={<PackagePlus size={15} />}
            onClick={openSave}
            sx={noWrap}
          >{t`Save as bundle…`}</Button>
          <Button
            size="small"
            variant="outlined"
            color="error"
            disabled={locked}
            startIcon={<Trash2 size={15} />}
            onClick={() => askRemove(selected)}
            sx={noWrap}
          >{t`Remove`}</Button>
        </Box>
      </DisabledReason>
      <Button
        size="small"
        variant="outlined"
        startIcon={<Share2 size={15} />}
        onClick={share}
        sx={noWrap}
      >{t`Share selection`}</Button>
      <Button
        size="small"
        variant="text"
        startIcon={<X size={15} />}
        onClick={clear}
        sx={noWrap}
      >{t`Clear`}</Button>
    </Box>
  )
}
