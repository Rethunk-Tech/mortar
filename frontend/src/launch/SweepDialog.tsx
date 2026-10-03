import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from '@mui/material'
import { useState } from 'react'
import type { SweepReport } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launchsvc/models.ts'
import { UpdateEverywhere } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { paper } from '../mods/paper.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useSweepUi } from './events.ts'

const DIALOG_WIDTH = 480
const EDGE = 32

function fixLabel(i18n: I18n, fix: string): string {
  if (fix === 'update') {
    return i18n._(msg`Update`)
  }
  if (fix === 'off') {
    return i18n._(msg`Turn off`)
  }
  return i18n._(msg`No fix yet`)
}

function fixableIds(report: SweepReport): string[] {
  const ids: string[] = []
  for (const row of report.profiles ?? []) {
    for (const mod of row.broken ?? []) {
      if (mod.fix === 'update') {
        ids.push(mod.uniqueId || mod.key)
      }
    }
  }
  return [...new Set(ids.filter((id) => id !== ''))]
}

function ProfileRows({ report, i18n }: { report: SweepReport; i18n: I18n }) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      {(report.profiles ?? []).map((row) => (
        <Box key={row.profile}>
          <Typography sx={{ fontSize: 13, fontWeight: 600 }}>{row.name}</Typography>
          {(row.broken ?? []).map((mod) => (
            <Typography key={`${row.profile}-${mod.key}`} sx={{ fontSize: 13 }}>
              {i18n._(msg`${mod.name}: ${fixLabel(i18n, mod.fix)}`)}
            </Typography>
          ))}
          {(row.missingDeps ?? 0) > 0 ? (
            <Typography sx={{ fontSize: 13 }} color="text.secondary">
              {i18n._(msg`${row.missingDeps} missing dependencies`)}
            </Typography>
          ) : null}
        </Box>
      ))}
    </Box>
  )
}

function SweepDialog() {
  const { i18n, t } = useLingui()
  const report = useSweepUi((s) => s.report)
  const open = useSweepUi((s) => s.open)
  const close = useSweepUi((s) => s.close)
  const [pending, setPending] = useState(false)
  const ids = report ? fixableIds(report) : []
  const apply = () => {
    if (!report) {
      return
    }
    setPending(true)
    Promise.all(ids.map((id) => UpdateEverywhere(report.game, id, 'latest')))
      .then(() => {
        useToasts.getState().push({ kind: 'success', title: t`Updated fixable mods` })
        close()
      })
      .catch(reportUnexpected)
      .finally(() => setPending(false))
  }
  return (
    <Dialog
      open={open && report !== null}
      onClose={pending ? undefined : close}
      transitionDuration={0}
      slotProps={{
        paper: { sx: { ...paper.sx, width: DIALOG_WIDTH, maxWidth: `calc(100% - ${EDGE}px)` } },
      }}
    >
      <DialogTitle>{t`Patch-day review`}</DialogTitle>
      <DialogContent>{report ? <ProfileRows report={report} i18n={i18n} /> : null}</DialogContent>
      <DialogActions sx={{ bgcolor: 'background.paper' }}>
        <Button onClick={close} disabled={pending}>
          {t`Close`}
        </Button>
        <Button variant="contained" onClick={apply} disabled={pending || ids.length === 0}>
          {t`Update all fixable`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export { SweepDialog }
