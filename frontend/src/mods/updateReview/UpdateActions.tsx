import { useLingui } from '@lingui/react/macro'
import { Button, IconButton, Menu, MenuItem } from '@mui/material'
import { MoreHorizontal } from 'lucide-react'
import { useState } from 'react'
import type { Update } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import type {
  Mod,
  Profile,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { download } from '../../queue/actions.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { useMods } from '../store.ts'
import { downloadable, updateWant } from './wants.ts'

export function UpdateActions({
  update,
  mod,
  entry,
  queued,
  caution,
  acked,
  onUpdateAll,
}: {
  update: Update
  mod?: Mod
  entry?: NonNullable<Profile['entries']>[number]
  queued: boolean
  caution: string
  acked: boolean
  onUpdateAll: () => void
}) {
  const { t } = useLingui()
  const setPinned = useMods((s) => s.setPinned)
  const setSkipVersion = useMods((s) => s.setSkipVersion)
  const setSkipSource = useMods((s) => s.setSkipSource)
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const blocked = queued || (caution !== '' && !acked)
  const act = (fn: () => Promise<unknown>) => () => {
    setAnchor(null)
    fn().catch(reportUnexpected)
  }
  return (
    <>
      {downloadable(update) ? (
        <Button
          variant="contained"
          disabled={blocked}
          onClick={() => download([updateWant(update)]).catch(reportUnexpected)}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {queued ? t`Queued` : t`Update`}
        </Button>
      ) : null}
      <IconButton
        aria-label={t`More actions for ${update.name}`}
        aria-haspopup="menu"
        onClick={(e) => setAnchor(e.currentTarget)}
        sx={{ borderRadius: '6px', bgcolor: anchor ? 'rgba(255,255,255,0.1)' : 'transparent' }}
      >
        <MoreHorizontal size={18} />
      </IconButton>
      <Menu
        open={anchor !== null}
        anchorEl={anchor}
        onClose={() => setAnchor(null)}
        transitionDuration={0}
      >
        {downloadable(update) ? (
          <MenuItem disabled={blocked} onClick={act(async () => onUpdateAll())}>
            {t`Update in all profiles that have it`}
          </MenuItem>
        ) : null}
        <MenuItem
          disabled={!mod}
          onClick={act(async () => mod && setSkipVersion(mod, update.version))}
        >
          {t`Skip this version`}
        </MenuItem>
        <MenuItem disabled={!mod} onClick={act(async () => mod && setPinned(mod, !entry?.pinned))}>
          {entry?.pinned ? t`Unpin` : t`Pin`}
        </MenuItem>
        {update.source && !entry?.skipSources?.includes(update.source) ? (
          <MenuItem
            disabled={!mod}
            onClick={act(async () => mod && setSkipSource(mod, update.source, true))}
          >
            {t`Ignore updates from ${update.source}`}
          </MenuItem>
        ) : null}
      </Menu>
    </>
  )
}
