import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import type { HostValue } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/configsvc/models.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { configApi } from './api.ts'
import { useTypedConfig } from './store.ts'

/** Keeps the host's values of the open file's shared settings as the player's own, written once the game closes. */
export function SaveHostValues({ host }: { host: Map<string, HostValue> }) {
  const { t } = useLingui()
  const target = useTypedConfig((s) => s.target)
  const file = useTypedConfig((s) => s.current)
  if (host.size === 0 || !target || !file) {
    return null
  }
  const save = () => {
    configApi
      .saveHostValues(target, file)
      .then(() =>
        useToasts.getState().push({
          kind: 'success',
          title: t`Saved the host's values`,
          body: t`They are written to your config when the game closes.`,
        }),
      )
      .catch(reportUnexpected)
  }
  return (
    <Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
      <Button size="small" variant="outlined" onClick={save}>
        {t`Save host's values`}
      </Button>
    </Box>
  )
}
