import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import type { Ref } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { useProfiles } from '../../profiles/store.ts'
import { download, type Want } from '../../queue/actions.ts'
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
      sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
    >
      {t`Open page`}
    </Button>
  )
  const github = where.site === 'GitHub' && where.github !== ''
  if (!github && (where.site !== 'Nexus' || where.pageId <= 0)) {
    return open
  }
  const queued = github
    ? pendingFor(queue, profileId, 0, where.github)
    : pendingFor(queue, profileId, where.pageId)
  const want: Want = github
    ? { kind: 'dependency', repo: where.github, name: where.github }
    : {
        kind: 'dependency',
        modId: where.pageId,
        latest: true,
        fileId: where.fileId,
        name: where.pageName,
        fileName: where.fileName,
        version: where.version,
      }
  return (
    <>
      {open}
      <Button
        size="small"
        variant="contained"
        color="warning"
        disabled={queued}
        onClick={() => download([want]).catch(reportUnexpected)}
        sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
      >
        {queued ? t`Queued` : (addLabel ?? '')}
      </Button>
    </>
  )
}
