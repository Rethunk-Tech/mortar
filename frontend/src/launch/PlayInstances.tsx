import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { PrefNumber } from '../settings/PrefControls.tsx'
import { specByKey, usePrefSpecs } from '../settings/prefSpecs.ts'
import { prefAsNumber, prefRaw } from '../settings/prefValue.ts'
import { useSettings } from '../settings/store.ts'
import { space } from '../theme/density.ts'

// Several copies start only with a Direct launch; through a store the fields show why they are off.
export function PlayInstances({ game, direct }: { game: string; direct: boolean }) {
  const { t } = useLingui()
  const specs = usePrefSpecs()
  const instances = specByKey(specs, 'instances')
  const delay = specByKey(specs, 'delaySeconds')
  const instancesRaw = useSettings((s) => (instances ? prefRaw(s, instances, game) : undefined))
  const delayRaw = useSettings((s) => (delay ? prefRaw(s, delay, game) : undefined))
  if (!(instances && delay)) {
    return null
  }
  return (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: 'auto 1fr',
        alignItems: 'center',
        gap: space.gap,
        px: space.pad,
        py: space.gap,
      }}
    >
      <Typography sx={{ fontSize: 12, color: 'var(--mortar-ink-soft)' }}>{t`Instances`}</Typography>
      <PrefNumber
        value={prefAsNumber(instancesRaw, instances, game)}
        min={instances.min ?? 1}
        max={instances.max || 8}
        label={t`Instances`}
        disabled={!direct}
        onCommit={(n) => SetByKey('instances', String(n), game)}
      />
      <Typography sx={{ fontSize: 12, color: 'var(--mortar-ink-soft)' }}>{t`Delay (s)`}</Typography>
      <PrefNumber
        value={prefAsNumber(delayRaw, delay, game)}
        min={delay.min ?? 1}
        max={delay.max || 60}
        label={t`Delay seconds`}
        disabled={!direct}
        onCommit={(n) => SetByKey('delaySeconds', String(n), game)}
      />
      {direct ? null : (
        <Typography sx={{ gridColumn: '1 / -1', fontSize: 12, color: 'var(--mortar-ink-soft)' }}>
          {t`Several copies start only with Direct launch.`}
        </Typography>
      )}
    </Box>
  )
}
