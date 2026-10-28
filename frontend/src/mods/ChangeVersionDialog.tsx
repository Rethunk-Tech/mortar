import { useLingui } from '@lingui/react/macro'
import { Box, Button, Chip, List, ListItemButton, Typography } from '@mui/material'
import { History } from 'lucide-react'
import { useState } from 'react'
import { formatWhen } from '../i18n/formatWhen.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { AppDialog, DialogFooter, ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { ErrorRetry } from '../shell/ErrorRetry.tsx'
import { SkeletonRows } from '../shell/SkeletonRows.tsx'
import { space } from '../theme/density.ts'
import { isFrozen } from './frozen.ts'
import { useListedVersions } from './useListedVersions.ts'
import {
  isInstalledVersion,
  type VersionChoice,
  type VersionTarget,
  versionChoices,
} from './versionChoices.ts'

// The list keeps one height while it loads and once it has arrived, so the dialog does not jump.
const LIST_PX = 320
const ROW_PX = 44
const SKELETON_ROWS = 6

/**
 * Picks one of the versions a site lists for a mod and queues it. With an entry (key, its version and whether it is
 * held), the choice replaces that mod, older or newer; a held mod asks again first, because a freeze or pin exists to
 * keep its version.
 */
export function ChangeVersionDialog({
  game,
  name,
  target,
  entry,
  onClose,
}: {
  game: string
  name: string
  target: VersionTarget
  entry?: { key: string; version: string; pinned: boolean }
  onClose: () => void
}) {
  const { t } = useLingui()
  const listed = useListedVersions(game, target.source, target.id)
  const frozen = isFrozen(useProfiles(openProfileOf))
  const [picked, setPicked] = useState('')
  const [asking, setAsking] = useState(false)
  const installed = entry?.version ?? ''
  const choices: VersionChoice[] =
    listed.state === 'ready' ? versionChoices(target, listed.versions, entry?.key ?? '') : []
  const choice = choices.find((c) => c.key === picked)
  const held = entry?.pinned === true
  const go = () => {
    if (choice) {
      onClose()
      download([choice.want], true).catch(() => undefined)
    }
  }
  let confirmLabel = choice ? t`Install version ${{ version: choice.version }}` : t`Install`
  if (asking) {
    confirmLabel = t`Install anyway`
  }
  const body = (() => {
    if (asking && choice) {
      return (
        <Typography>
          {frozen
            ? t`This profile is frozen, so ${name} is held at ${installed}. Install ${{ version: choice.version }} anyway? The mod stays held at the new version.`
            : t`${name} is pinned at ${installed}. Install ${{ version: choice.version }} anyway? The pin stays on the new version.`}
        </Typography>
      )
    }
    if (listed.state === 'loading') {
      return <SkeletonRows label={t`Reading the versions…`} count={SKELETON_ROWS} height={ROW_PX} />
    }
    if (listed.state === 'error') {
      return <ErrorRetry error={listed.error} onRetry={listed.retry} />
    }
    if (choices.length === 0) {
      return <Typography sx={{ color: 'text.secondary' }}>{t`No versions are listed.`}</Typography>
    }
    return (
      <List aria-label={t`Versions`} disablePadding={true} sx={{ overflowY: 'auto', mx: -1 }}>
        {choices.map((c, i) => {
          const here = isInstalledVersion(c, installed)
          return (
            <ListItemButton
              key={c.key}
              selected={c.key === picked}
              aria-current={here ? 'true' : undefined}
              onClick={() => setPicked(here ? '' : c.key)}
              sx={{ minHeight: ROW_PX, gap: space.gap, borderRadius: 1, px: 1 }}
            >
              <Box sx={{ flex: 1, minWidth: 0 }}>
                <Typography sx={{ fontSize: 'base.fontSize', overflowWrap: 'anywhere' }}>
                  {c.label}
                </Typography>
                {c.released ? (
                  <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>
                    {formatWhen(c.released)}
                  </Typography>
                ) : null}
              </Box>
              {i === 0 ? <Chip size="small" variant="outlined" label={t`Latest`} /> : null}
              {here ? <Chip size="small" label={t`Installed`} /> : null}
            </ListItemButton>
          )
        })}
      </List>
    )
  })()
  return (
    <AppDialog
      open={true}
      onClose={onClose}
      size="sm"
      title={entry ? t`Change the version of ${name}` : t`Install a version of ${name}`}
      icon={
        <ToneIcon tone="neutral">
          <History size={22} />
        </ToneIcon>
      }
      footer={
        <DialogFooter aside={choice || asking ? null : t`Pick a version.`}>
          <Button onClick={asking ? () => setAsking(false) : onClose}>
            {asking ? t`Back` : t`Cancel`}
          </Button>
          <Button
            variant="contained"
            disabled={!choice}
            onClick={held && !asking ? () => setAsking(true) : go}
          >
            {confirmLabel}
          </Button>
        </DialogFooter>
      }
    >
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: space.gap,
          height: asking ? undefined : LIST_PX,
        }}
      >
        {body}
      </Box>
    </AppDialog>
  )
}
