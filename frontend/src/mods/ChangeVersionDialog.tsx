import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Radio,
  RadioGroup,
  Typography,
} from '@mui/material'
import { useState } from 'react'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { ErrorRetry } from '../shell/ErrorRetry.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { space } from '../theme/density.ts'
import { isFrozen } from './frozen.ts'
import { useListedVersions } from './useListedVersions.ts'
import {
  isInstalledVersion,
  type VersionChoice,
  type VersionTarget,
  versionChoices,
} from './versionChoices.ts'

const LIST_MAX_PX = 320

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
      return <LoadingRow>{t`Reading the versions…`}</LoadingRow>
    }
    if (listed.state === 'error') {
      return <ErrorRetry error={listed.error} onRetry={listed.retry} />
    }
    if (choices.length === 0) {
      return <Typography sx={{ color: 'text.secondary' }}>{t`No versions are listed.`}</Typography>
    }
    return (
      <RadioGroup
        value={picked}
        onChange={(_, v) => setPicked(v)}
        sx={{ maxHeight: LIST_MAX_PX, overflowY: 'auto', flexWrap: 'nowrap' }}
      >
        {choices.map((c, i) => {
          const here = isInstalledVersion(c, installed)
          const tags = [i === 0 ? t`latest` : '', here ? t`Installed` : ''].filter(Boolean)
          return (
            <FormControlLabel
              key={c.key}
              value={c.key}
              disabled={here}
              control={<Radio size="small" />}
              label={tags.length > 0 ? `${c.label} (${tags.join(', ')})` : c.label}
            />
          )
        })}
      </RadioGroup>
    )
  })()
  return (
    <Dialog open={true} onClose={onClose} fullWidth={true} maxWidth="xs">
      <DialogTitle>
        {entry ? t`Change the version of ${name}` : t`Install a version of ${name}`}
      </DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap, pt: 0.5 }}>{body}</Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={asking ? () => setAsking(false) : onClose}>
          {asking ? t`Back` : t`Cancel`}
        </Button>
        <Button
          variant="contained"
          disabled={!choice}
          onClick={held && !asking ? () => setAsking(true) : go}
        >
          {asking ? t`Install anyway` : t`Install`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
