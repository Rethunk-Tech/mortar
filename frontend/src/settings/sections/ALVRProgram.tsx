import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { PickExecutable } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { ALVR } from '../../brand/productNames.ts'
import { space } from '../../theme/density.ts'
import { pickSetting } from '../persist.ts'
import { prefCopy } from '../prefCopy.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

/** The ALVR dashboard Play runs first; one program for every game, shown where a game is set to start ALVR. */
export function ALVRProgram() {
  const { t, i18n } = useLingui()
  const program = useSettings((s) => s.alvrProgram ?? '')
  const copy = prefCopy(i18n, 'alvrProgram')
  const choose = () =>
    pickSetting(() => PickExecutable(t`Choose the ${{ alvr: ALVR }} dashboard`), 'alvrProgram', '')
  return (
    <SettingRow label={copy.label} description={copy.description ?? ''}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, minWidth: 0 }}>
        <Typography
          noWrap={true}
          title={program}
          color={program ? undefined : 'text.secondary'}
          sx={{ maxWidth: 360 }}
        >
          {program || t`Not chosen`}
        </Typography>
        <Button variant="outlined" onClick={choose} sx={{ flexShrink: 0 }}>
          {t`Choose…`}
        </Button>
      </Box>
    </SettingRow>
  )
}
