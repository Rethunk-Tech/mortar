import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import type { Ref } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { useProfiles } from '../../profiles/store.ts'
import { download, refWant } from '../../queue/actions.ts'
import { useQueue } from '../../queue/store.ts'
import { pendingFor } from '../../queue/totals.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { openPage } from '../menu.ts'

export function WhereButtons({ where, addLabel }: { where: Ref; addLabel: string }) {
  const { t } = useLingui()
  const queue = useQueue((s) => s.state.items)
  const profileId = useProfiles((s) => s.openId)
  const { url } = where
  if (!url) {
    return null
  }
  const open = (
    <Button
      size="small"
      color="warning"
      variant="outlined"
      onClick={() => openPage(url)}
      sx={{ flexShrink: 0 }}
    >
      {t`Open page`}
    </Button>
  )
  const want = refWant(where, 'dependency')
  if (!want) {
    return open
  }
  const queued = pendingFor(queue, profileId, want.modId ?? 0, want.repo)
  return (
    <>
      {open}
      <Button
        size="small"
        variant="contained"
        color="warning"
        disabled={queued}
        onClick={() => download([want]).catch(reportUnexpected)}
        sx={{ flexShrink: 0 }}
      >
        {queued ? t`Queued` : addLabel}
      </Button>
    </>
  )
}
