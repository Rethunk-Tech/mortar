import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
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
  return (
    <>
      <Button
        size="small"
        disabled={!mod}
        onClick={() => mod && setSkipVersion(mod, update.version).catch(reportUnexpected)}
        sx={{ whiteSpace: 'nowrap' }}
      >
        {t`Skip this version`}
      </Button>
      <Button
        size="small"
        onClick={() => mod && setPinned(mod, !entry?.pinned).catch(reportUnexpected)}
        sx={{ whiteSpace: 'nowrap' }}
      >
        {entry?.pinned ? t`Unpin` : t`Pin`}
      </Button>
      {update.source && !entry?.skipSources?.includes(update.source) ? (
        <Button
          size="small"
          disabled={!mod}
          onClick={() => mod && setSkipSource(mod, update.source, true).catch(reportUnexpected)}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t`Ignore updates from ${update.source}`}
        </Button>
      ) : null}
      {downloadable(update) ? (
        <>
          <Button
            variant="contained"
            disabled={queued || (caution !== '' && !acked)}
            onClick={() => download([updateWant(update)]).catch(reportUnexpected)}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {queued ? t`Queued` : t`Update`}
          </Button>
          <Button
            variant="outlined"
            disabled={queued || (caution !== '' && !acked)}
            onClick={onUpdateAll}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {t`Update in all profiles that have it`}
          </Button>
        </>
      ) : null}
    </>
  )
}
