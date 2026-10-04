import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, LinearProgress, Typography } from '@mui/material'
import { Events } from '@wailsio/runtime'
import { useEffect, useState } from 'react'
import type {
  Progress,
  Summary,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/storecheck/models.ts'
import {
  Check,
  CheckProgress,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/storecheck/service.ts'
import { usePending } from '../../toasts/usePending.ts'
import { SettingRow } from '../SettingsSection.tsx'

const FULL = 100
const IDLE: Progress = { running: false, done: 0, total: 0, damaged: 0 }

function useCheckProgress() {
  const [progress, setProgress] = useState<Progress>(IDLE)
  useEffect(() => {
    CheckProgress()
      .then(setProgress)
      .catch(() => setProgress(IDLE))
    return Events.On('storecheck:progress', (event) => setProgress(event.data as Progress))
  }, [])
  return progress
}

function Result({ summary }: { summary: Summary }) {
  const { t } = useLingui()
  const damaged = summary.damaged ?? []
  const checked = plural(summary.checked, { one: '# mod checked', other: '# mods checked' })
  if (damaged.length === 0) {
    return (
      <Typography sx={{ fontSize: 14, color: 'text.secondary' }}>
        {summary.failed > 0
          ? t`${checked}. ${plural(summary.failed, { one: '# could not be read.', other: '# could not be read.' })}`
          : t`${checked}. All files are intact.`}
      </Typography>
    )
  }
  const bad = plural(damaged.length, { one: '# has damaged files', other: '# have damaged files' })
  return (
    <Box sx={{ fontSize: 14 }}>
      <Typography
        sx={{ fontSize: 14 }}
      >{t`${checked}; ${bad}. Repair them from each profile's Problems tab.`}</Typography>
      <Box component="ul" sx={{ m: 0, mt: 0.5, pl: 2.5, color: 'text.secondary' }}>
        {damaged.map((item) => (
          <li key={`${item.game}/${item.key}`}>
            {t`${item.name}: ${item.missing} missing, ${item.changed} changed, ${item.extra} extra`}
          </li>
        ))}
      </Box>
    </Box>
  )
}

export function StoreCheckRow() {
  const { t } = useLingui()
  const progress = useCheckProgress()
  const [summary, setSummary] = useState<Summary | null>(null)
  const [pending, run] = usePending()
  const running = pending || progress.running
  const start = () => {
    setSummary(null)
    run(async () => setSummary(await Check()), { errorTitle: t`Could not check the store files` })
  }
  const percent = progress.total === 0 ? 0 : (FULL * progress.done) / progress.total
  return (
    <SettingRow
      label={t`Check store files`}
      description={t`Compares every stored mod's files with what was saved when it was added. Mortar also does this slowly in the background, about once a week.`}
      block={true}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
        <Button variant="outlined" disabled={running} onClick={start}>
          {t`Check now`}
        </Button>
        {running ? (
          <Box sx={{ flex: 1, display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <LinearProgress variant="determinate" value={percent} sx={{ flex: 1 }} />
            <Typography sx={{ fontSize: 14, fontVariantNumeric: 'tabular-nums' }}>
              {t`${progress.done} of ${progress.total}`}
            </Typography>
          </Box>
        ) : null}
      </Box>
      {summary === null ? null : <Result summary={summary} />}
    </SettingRow>
  )
}
