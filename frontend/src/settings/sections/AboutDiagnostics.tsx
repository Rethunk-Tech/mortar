import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { CircleCheck, CircleX, Copy, TriangleAlert } from 'lucide-react'
import { useState } from 'react'
import type { Report } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/doctor/models.ts'
import { Doctor } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/support/service.ts'
import { errorText, reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'

const detail = { color: 'rgba(235,235,240,0.95)' }

function StatusIcon({ status }: { status: string }) {
  if (status === 'fail') {
    return <CircleX size={16} color="#C70A0A" aria-hidden={true} />
  }
  if (status === 'warn') {
    return <TriangleAlert size={16} color="#F3B416" aria-hidden={true} />
  }
  return <CircleCheck size={16} color="#0CDF64" aria-hidden={true} />
}

export function Diagnostics() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const [report, setReport] = useState<Report | null>(null)
  const [busy, setBusy] = useState(false)
  const run = () => {
    setBusy(true)
    Doctor()
      .then((r) => setReport(r ?? { checks: [] }))
      .catch(reportUnexpected)
      .finally(() => setBusy(false))
  }
  const copy = () => {
    if (!report) {
      return
    }
    const text =
      (report.checks ?? []).map((c) => c.detail).join('\n') + (report.checks?.length ? '\n' : '')
    navigator.clipboard.writeText(text).then(
      () => push({ kind: 'success', title: t`Copied the report` }),
      (err: unknown) => {
        const msg = errorText(err)
        push({ kind: 'error', title: t`Could not copy the report`, ...(msg ? { body: msg } : {}) })
      },
    )
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
      <Box sx={{ fontWeight: 600 }}>{t`Diagnostics`}</Box>
      <Box sx={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
        <Button
          variant="outlined"
          color="inherit"
          disabled={busy}
          onClick={run}
          sx={{ alignSelf: 'flex-start', whiteSpace: 'nowrap' }}
        >
          {t`Run checks`}
        </Button>
        <Button
          variant="outlined"
          color="inherit"
          startIcon={<Copy size={16} />}
          disabled={!report}
          onClick={copy}
          sx={{ alignSelf: 'flex-start', whiteSpace: 'nowrap' }}
        >
          {t`Copy report`}
        </Button>
      </Box>
      {(report?.checks ?? []).map((c) => (
        <Box key={c.id} sx={{ display: 'flex', gap: '8px', alignItems: 'flex-start', minWidth: 0 }}>
          <StatusIcon status={c.status} />
          <Box component="span" sx={{ ...detail, minWidth: 0 }}>
            {c.detail}
          </Box>
        </Box>
      ))}
    </Box>
  )
}
