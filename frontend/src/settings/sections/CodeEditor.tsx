import { useLingui } from '@lingui/react/macro'
import { Box, MenuItem, Select } from '@mui/material'
import { useLoaded } from '@rethunk/react-load-hooks'
import { Editors } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { PickExecutable } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { SetByKey } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { pickSetting, saveSetting } from '../persist.ts'
import { prefCopy } from '../prefCopy.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

const SYSTEM = 'system'
// Picks a program rather than being one, so it is never saved.
const OTHER = 'other'

const PATH_SEPARATOR = /[\\/]/
const fileName = (path: string) => path.split(PATH_SEPARATOR).pop() ?? path

/** Where Open in editor and a compiler error open: whichever is found first, the default app, or one program. */
export function CodeEditor() {
  const { t, i18n } = useLingui()
  const copy = prefCopy(i18n, 'codeEditor')
  const chosen = useSettings((s) => s.codeEditor ?? '')
  const { data } = useLoaded(() => Editors(), [], null)
  const editors = data ?? []
  const known = chosen === '' || chosen === SYSTEM || editors.some((e) => e.path === chosen)
  const listed = known ? editors : [...editors, { name: fileName(chosen), path: chosen }]
  const set = (value: string) => saveSetting(() => SetByKey('codeEditor', value, ''))
  const pick = () => pickSetting(() => PickExecutable(t`Choose a code editor`), 'codeEditor', '')
  const automatic = t`Automatic`
  const system = t`Default app for the project`
  const nameOf = (value: unknown) => {
    if (value === '') {
      return automatic
    }
    if (value === SYSTEM) {
      return system
    }
    return listed.find((e) => e.path === value)?.name ?? String(value)
  }
  return (
    <SettingsSection title={t`Developing mods`}>
      <SettingRow label={copy.label} description={copy.description ?? ''}>
        <Select
          size="small"
          displayEmpty={true}
          value={chosen}
          renderValue={nameOf}
          SelectDisplayProps={{ 'aria-label': copy.label } as object}
          onChange={(e) => {
            const value = String(e.target.value)
            if (value === OTHER) {
              pick()
            } else {
              set(value)
            }
          }}
          sx={{ minWidth: 240, maxWidth: 360 }}
        >
          <MenuItem value="">{automatic}</MenuItem>
          <MenuItem value={SYSTEM}>{system}</MenuItem>
          {listed.map((e) => (
            <MenuItem key={e.path} value={e.path} title={e.path}>
              <Box sx={{ minWidth: 0 }}>
                <div>{e.name}</div>
                <Box
                  sx={{
                    maxWidth: 480,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    fontSize: 'small.fontSize',
                    color: 'text.secondary',
                  }}
                >
                  {e.path}
                </Box>
              </Box>
            </MenuItem>
          ))}
          <MenuItem value={OTHER}>{t`Other program…`}</MenuItem>
        </Select>
      </SettingRow>
    </SettingsSection>
  )
}
