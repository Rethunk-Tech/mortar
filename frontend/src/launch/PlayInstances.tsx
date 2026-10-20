import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import type { ReactNode } from 'react'
import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { PrefNumber } from '../settings/PrefControls.tsx'
import { specByKey, usePrefSpecs } from '../settings/prefSpecs.ts'
import { prefAsNumber, prefRaw } from '../settings/prefValue.ts'
import { useSettings } from '../settings/store.ts'
import { MenuHeading } from '../shell/TitleMenu.tsx'
import { reportError } from '../toasts/report.ts'

const FIELD_PX = 96
const INSET_PX = 16
const NOTE_MAX_PX = 260

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Typography sx={{ fontSize: 12, color: 'var(--mortar-ink-soft)' }}>{label}</Typography>
      {children}
    </Box>
  )
}

// How many copies of the game Play starts, and how far apart. Several copies start only with a Direct launch; through
// a store the section says so and offers to switch.
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
    <Box sx={{ pb: 1 }}>
      <MenuHeading>{t`Copies of the game`}</MenuHeading>
      {direct ? (
        <Box sx={{ display: 'flex', gap: 1.5, px: `${INSET_PX}px`, pt: 0.5 }}>
          <Field label={t`Copies`}>
            <PrefNumber
              value={prefAsNumber(instancesRaw, instances, game)}
              min={instances.min ?? 1}
              max={instances.max || 8}
              label={t`Instances`}
              width={FIELD_PX}
              onCommit={(n) => SetByKey('instances', String(n), game)}
            />
          </Field>
          <Field label={t`Delay between`}>
            <PrefNumber
              value={prefAsNumber(delayRaw, delay, game)}
              min={delay.min ?? 1}
              max={delay.max || 60}
              label={t`Delay seconds`}
              width={FIELD_PX}
              suffix={t`s`}
              onCommit={(n) => SetByKey('delaySeconds', String(n), game)}
            />
          </Field>
        </Box>
      ) : (
        <Box
          sx={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'flex-start',
            gap: 1,
            px: `${INSET_PX}px`,
            pt: 0.5,
            maxWidth: NOTE_MAX_PX,
          }}
        >
          <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
            {t`This game starts through its store, which runs one copy. A Direct launch can start several.`}
          </Typography>
          <Button
            size="small"
            variant="outlined"
            onClick={() =>
              SetByKey('defaultLaunchMethod', 'direct', game).catch(
                reportError(t`Could not save that setting`),
              )
            }
          >
            {t`Use Direct launch`}
          </Button>
        </Box>
      )}
    </Box>
  )
}
