import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button, LinearProgress, Typography } from '@mui/material'
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
import { space } from '../../theme/density.ts'
import { usePending } from '../../toasts/usePending.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettingsRelevant } from '../useSettingsSearch.ts'

const FULL = 100
const IDLE: Progress = { running: false, done: 0, total: 0, damaged: 0 }

function useCheckProgress(relevant: boolean) {
  const [progress, setProgress] = useState<Progress>(IDLE)
  useEffect(() => {
    if (!relevant) {
      return
    }
    CheckProgress()
      .then(setProgress)
      .catch(() => setProgress(IDLE))
    return Events.On('storecheck:progress', (event) => setProgress(event.data as Progress))
  }, [relevant])
  return progress
}

function Result({ summary }: { summary: Summary }) {
  const { t } = useLingui()
  const damaged = summary.damaged ?? []
  const checked = plural(
    { count: summary.checked },
    { one: '# mod checked', other: '# mods checked' },
  )
  if (damaged.length === 0) {
    return summary.failed > 0 ? (
      <Alert variant="compact" severity="warning">
        {t`${checked}. ${plural({ count: summary.failed }, { one: '# could not be read.', other: '# could not be read.' })}`}
      </Alert>
    ) : (
      <Alert variant="compact" severity="success">
        {t`${checked}. All files are intact.`}
      </Alert>
    )
  }
  const bad = plural(
    { count: damaged.length },
    { one: '# has damaged files', other: '# have damaged files' },
  )
  return (
    <Alert variant="compact" severity="error">
      {t`${checked}; ${bad}. Repair them from each profile's Problems tab.`}
      <Box component="ul" sx={{ m: 0, mt: 0.5, pl: space.pad }}>
        {damaged.map((item) => (
          <li key={`${item.game}/${item.key}`}>
            {t`${{ name: item.name }}: ${{ missing: item.missing }} missing, ${{ changed: item.changed }} changed, ${{ extra: item.extra }} extra`}
          </li>
        ))}
      </Box>
    </Alert>
  )
}

export function StoreCheckRow() {
  const { t } = useLingui()
  const label = t`Check store files`
  const text = t`Compares every stored mod's files with what was saved when it was added. Mortar also runs it slowly in the background, about weekly.`
  const progress = useCheckProgress(useSettingsRelevant(label, text))
  const [summary, setSummary] = useState<Summary | null>(null)
  const [pending, run] = usePending()
  const running = pending || progress.running
  const start = () => {
    setSummary(null)
    run(async () => setSummary(await Check()), { errorTitle: t`Could not check the store files` })
  }
  const percent = progress.total === 0 ? 0 : (FULL * progress.done) / progress.total
  return (
    <SettingRow label={label} description={text} block={true}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.pad }}>
        <Button variant="outlined" disabled={running} onClick={start}>
          {t`Check now`}
        </Button>
        {running ? (
          <Box sx={{ flex: 1, display: 'flex', alignItems: 'center', gap: space.gap }}>
            <LinearProgress variant="determinate" value={percent} sx={{ flex: 1 }} />
            <Typography sx={{ fontSize: 'medium.fontSize', fontVariantNumeric: 'tabular-nums' }}>
              {t`${{ done: progress.done }} of ${{ total: progress.total }}`}
            </Typography>
          </Box>
        ) : null}
      </Box>
      {summary === null ? null : <Result summary={summary} />}
    </SettingRow>
  )
}
