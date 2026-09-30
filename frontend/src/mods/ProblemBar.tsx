import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { Browser } from '@wailsio/runtime'
import { TriangleAlert } from 'lucide-react'
import { useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { useQueue } from '../queue/store.ts'
import { pendingFor } from '../queue/totals.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useDescribe } from './describe.ts'
import { type Problem, problemsOf, sameId } from './lookup.ts'
import { useMods } from './store.ts'

const ROW_HEIGHT = 38
const ROW_GAP = 6
const VISIBLE_ROWS = 3

function FixButton({ problem }: { problem: Problem }) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const setEnabled = useMods((s) => s.setEnabled)
  const resolve = useMods((s) => s.resolve)
  const queue = useQueue((s) => s.state.items)
  const profileId = useProfiles((s) => s.openId)
  const button = (label: string, onClick: () => void) => (
    <Button
      size="small"
      variant="contained"
      color="warning"
      onClick={onClick}
      sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
    >
      {label}
    </Button>
  )
  if (problem.kind === 'duplicate') {
    return button(t`Resolve`, () => resolve(problem.duplicate))
  }
  if (problem.kind === 'broken') {
    const { broken } = problem
    const mod = mods.find((m) => m.key === broken.key && sameId(m.uniqueId, broken.uniqueId))
    return mod ? button(t`Switch off`, () => setEnabled(mod, false).catch(reportUnexpected)) : null
  }
  const { missing } = problem
  if (missing.reason === 'disabled') {
    const off = mods.find((m) => !m.enabled && sameId(m.uniqueId, missing.uniqueId))
    return off ? button(t`Switch on`, () => setEnabled(off, true).catch(reportUnexpected)) : null
  }
  const { where } = missing
  if (!where?.url) {
    return null
  }
  const { url } = where
  const open = (
    <Button
      size="small"
      color="warning"
      variant="outlined"
      onClick={() => Browser.OpenURL(url).catch(reportUnexpected)}
      sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
    >
      {t`Open page`}
    </Button>
  )
  if (where.site !== 'Nexus' || where.pageId <= 0) {
    return open
  }
  const queued = pendingFor(queue, profileId, where.pageId)
  return (
    <>
      {open}
      <Button
        size="small"
        variant="contained"
        color="warning"
        disabled={queued}
        onClick={() =>
          download([
            {
              kind: 'dependency',
              modId: where.pageId,
              fileId: where.fileId,
              name: where.pageName,
              fileName: where.fileName,
              version: where.version,
            },
          ]).catch(reportUnexpected)
        }
        sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
      >
        {queued ? t`Queued` : t`Add`}
      </Button>
    </>
  )
}

export function ProblemBar() {
  const { t } = useLingui()
  const describe = useDescribe()
  const result = useMods((s) => s.problems)
  const problems = problemsOf(result)
  if (problems.length === 0 && !result?.unknown) {
    return null
  }
  return (
    <Box sx={{ mx: 2, mt: 1.25, flexShrink: 0 }}>
      <Box
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: `${ROW_GAP}px`,
          maxHeight: VISIBLE_ROWS * ROW_HEIGHT + (VISIBLE_ROWS - 1) * ROW_GAP,
          overflowY: 'auto',
        }}
      >
        {problems.map((p) => (
          <Box
            key={JSON.stringify(p)}
            role="alert"
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: 1.25,
              minHeight: ROW_HEIGHT,
              flexShrink: 0,
              pl: 1.5,
              pr: 0.75,
              fontSize: 14,
              bgcolor: 'rgba(243,180,22,0.14)',
              border: '1px solid rgba(243,180,22,0.5)',
              borderRadius: '6px',
            }}
          >
            <Box component="span" sx={{ display: 'flex', flexShrink: 0, color: 'warning.main' }}>
              <TriangleAlert size={16} aria-hidden={true} />
            </Box>
            <Typography noWrap={true} sx={{ flex: 1, minWidth: 0, fontSize: 14 }}>
              {describe(p)}
            </Typography>
            <FixButton problem={p} />
          </Box>
        ))}
      </Box>
      {result?.unknown ? (
        <Typography sx={{ mt: 0.5, fontSize: 12, color: 'text.secondary' }}>
          {t`Some checks could not run without a connection, so more problems may show up later.`}
        </Typography>
      ) : null}
    </Box>
  )
}
