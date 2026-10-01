import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { Browser } from '@wailsio/runtime'
import { TriangleAlert } from 'lucide-react'
import { useProfiles } from '../profiles/store.ts'
import { download, type Want } from '../queue/actions.ts'
import { useQueue } from '../queue/store.ts'
import { pendingFor } from '../queue/totals.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useDescribe } from './describe.ts'
import { type Problem, problemsOf, sameId } from './lookup.ts'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

const ROW_HEIGHT = 38
const ROW_GAP = 6
const VISIBLE_ROWS = 3

function FixButton({ problem }: { problem: Problem }) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const setEnabled = useMods((s) => s.setEnabled)
  const resolve = useMods((s) => s.resolve)
  const dismissAsset = useMods((s) => s.dismissAsset)
  const queue = useQueue((s) => s.state.items)
  const profileId = useProfiles((s) => s.openId)
  const locked = useLocked()
  const button = (label: string, onClick: () => void) => (
    <Button
      size="small"
      variant="contained"
      color="warning"
      disabled={locked}
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
  if (problem.kind === 'asset') {
    const key = problem.asset.keys?.[0]
    const id = problem.asset.packIds?.[0]
    const mod = mods.find((m) => m.key === key && (id === undefined || sameId(m.uniqueId, id)))
    const dismiss = dismissAsset
    return (
      <>
        {mod ? button(t`Switch off`, () => setEnabled(mod, false).catch(reportUnexpected)) : null}
        {problem.asset.kind === 'edit' ? (
          <Button
            size="small"
            color="info"
            variant="outlined"
            onClick={() => dismiss(problem.asset).catch(reportUnexpected)}
            sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
          >
            {t`Dismiss`}
          </Button>
        ) : null}
      </>
    )
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
        {problems.map((p) => {
          const info = p.kind === 'asset' && p.asset.kind === 'edit'
          return (
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
                bgcolor: info ? 'rgba(56,189,248,0.12)' : 'rgba(243,180,22,0.14)',
                border: info ? '1px solid rgba(56,189,248,0.45)' : '1px solid rgba(243,180,22,0.5)',
                borderRadius: '6px',
              }}
            >
              <Box
                component="span"
                sx={{ display: 'flex', flexShrink: 0, color: info ? 'info.main' : 'warning.main' }}
              >
                <TriangleAlert size={16} aria-hidden={true} />
              </Box>
              <Typography noWrap={true} sx={{ flex: 1, minWidth: 0, fontSize: 14 }}>
                {describe(p)}
              </Typography>
              <FixButton problem={p} />
            </Box>
          )
        })}
      </Box>
      {result?.unknown ? (
        <Typography sx={{ mt: 0.5, fontSize: 12, color: 'text.secondary' }}>
          {t`Some checks could not run without a connection, so more problems may show up later.`}
        </Typography>
      ) : null}
    </Box>
  )
}
