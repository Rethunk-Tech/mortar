import { useLingui } from '@lingui/react/macro'
import { Button, IconButton, Menu, MenuItem } from '@mui/material'
import { MoreHorizontal } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type {
  EverywherePreview,
  Mod,
  Profile,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { PreviewEverywhere } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { openProfileOf, useProfiles } from '../../profiles/store.ts'
import { download } from '../../queue/actions.ts'
import { DisabledReason } from '../../shell/DisabledReason.tsx'
import { reportUnexpected } from '../../toasts/report.ts'
import { installableUpdate } from '../lookup.ts'
import { useNexusDetails } from '../nexusDetails.ts'
import { useOptionalSkips } from '../optionalFiles.ts'
import { useMods } from '../store.ts'
import { EverywhereDialog } from './EverywhereDialog.tsx'
import { withOptional } from './wants.ts'

export function UpdateActions({
  update,
  mod,
  entry,
  queued,
  caution,
  acked,
}: {
  update: Update
  mod?: Mod
  entry?: NonNullable<Profile['entries']>[number]
  queued: boolean
  caution: string
  acked: boolean
}) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const setPinned = useMods((s) => s.setPinned)
  const setSkipVersion = useMods((s) => s.setSkipVersion)
  const setSkipSource = useMods((s) => s.setSkipSource)
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [everywhere, setEverywhere] = useState(false)
  const [preview, setPreview] = useState<EverywherePreview | null>(null)
  useEffect(() => {
    if (game === '' || update.uniqueId === '') {
      return
    }
    PreviewEverywhere(game, update.uniqueId)
      .then(setPreview)
      .catch(() => setPreview(null))
  }, [game, update.uniqueId])
  const n = preview?.affected?.length ?? 0
  const previewReady = preview !== null
  const everywhereBlocked = !previewReady || n === 0
  const everywhereWhy = previewReady ? t`No eligible profiles.` : t`Checking profiles…`
  const missingWhy = t`This mod is not in the profile.`
  const blocked = queued || (caution !== '' && !acked)
  const act = (fn: () => Promise<unknown>) => () => {
    setAnchor(null)
    fn().catch(reportUnexpected)
  }
  return (
    <>
      {installableUpdate(update) ? (
        <Button
          variant="contained"
          disabled={blocked}
          onClick={() => {
            const { byId } = useNexusDetails.getState()
            const skipped = useOptionalSkips.getState().skipped[update.key] === true
            const profile = openProfileOf(useProfiles.getState())
            download(
              withOptional(update, profile, byId[update.nexusId]?.details?.files ?? [], skipped),
            ).catch(reportUnexpected)
          }}
        >
          {queued ? t`Queued` : t`Update`}
        </Button>
      ) : null}
      <IconButton
        aria-label={t`More actions for ${update.name}`}
        aria-haspopup="menu"
        aria-expanded={anchor !== null}
        onClick={(e) => setAnchor(e.currentTarget)}
        sx={{ borderRadius: '6px', bgcolor: anchor ? 'var(--mortar-hairline)' : 'transparent' }}
      >
        <MoreHorizontal size={18} />
      </IconButton>
      <Menu open={anchor !== null} anchorEl={anchor} onClose={() => setAnchor(null)}>
        {installableUpdate(update) ? (
          <DisabledReason title={everywhereWhy} disabled={everywhereBlocked}>
            <MenuItem
              disabled={everywhereBlocked || blocked}
              onClick={() => {
                setAnchor(null)
                setEverywhere(true)
              }}
            >
              {t`Update in all profiles (${n})`}
            </MenuItem>
          </DisabledReason>
        ) : null}
        <DisabledReason title={missingWhy} disabled={!mod}>
          <MenuItem
            disabled={!mod}
            onClick={act(async () => mod && setSkipVersion(mod, update.version))}
          >
            {t`Skip this update`}
          </MenuItem>
        </DisabledReason>
        <DisabledReason title={missingWhy} disabled={!mod}>
          <MenuItem
            disabled={!mod}
            onClick={act(async () => mod && setPinned(mod, !entry?.pinned))}
          >
            {entry?.pinned ? t`Unpin version` : t`Keep this version`}
          </MenuItem>
        </DisabledReason>
        {update.source && !entry?.skipSources?.includes(update.source) ? (
          <DisabledReason title={missingWhy} disabled={!mod}>
            <MenuItem
              disabled={!mod}
              onClick={act(async () => mod && setSkipSource(mod, update.source, true))}
            >
              {t`Ignore updates from ${update.source}`}
            </MenuItem>
          </DisabledReason>
        ) : null}
      </Menu>
      <EverywhereDialog
        open={everywhere}
        game={game}
        mods={[{ id: update.uniqueId, newKey: 'latest' }]}
        onClose={() => setEverywhere(false)}
      />
    </>
  )
}
