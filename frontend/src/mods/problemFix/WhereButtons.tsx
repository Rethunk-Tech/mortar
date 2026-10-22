import { useLingui } from '@lingui/react/macro'
import type { Ref } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import { useProfiles } from '../../profiles/store.ts'
import { download } from '../../queue/actions.ts'
import { refWant } from '../../queue/refWant.ts'
import { useQueue } from '../../queue/store.ts'
import { pendingFor } from '../../queue/totals.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { openPage } from '../menu.ts'
import { FixAction } from './fixActions.tsx'

export function WhereButtons({ where, addLabel }: { where: Ref; addLabel: string }) {
  const { t } = useLingui()
  const queue = useQueue((s) => s.state.items)
  const profileId = useProfiles((s) => s.openId)
  const { url } = where
  if (!url) {
    return null
  }
  const open = <FixAction label={t`Open page`} weight="other" onClick={() => openPage(url)} />
  const want = refWant(where, 'dependency')
  if (!want) {
    return open
  }
  const queued = pendingFor(queue, profileId, want.modId ?? 0, want.repo)
  return (
    <>
      {open}
      <FixAction
        label={queued ? t`Queued` : addLabel}
        weight="main"
        disabled={queued}
        onClick={() => download([want]).catch(reportUnexpected)}
      />
    </>
  )
}
