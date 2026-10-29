import { Box } from '@mui/material'
import type { ReactNode } from 'react'
import { space } from '../theme/density.ts'
import { PrefSelect } from './PrefControls.tsx'
import { ScopeDot, ScopeReset } from './PrefRow.tsx'
import { saveSetting } from './persist.ts'
import { SettingRow } from './SettingsSection.tsx'
import { useScopedKey } from './scopedKey.ts'

/** A hand-made select row for a game setting a profile can override, with the game's value beside the profile's. */
export function ScopedSelectRow({
  settingKey,
  game,
  gameValue,
  label,
  description,
  options,
}: {
  settingKey: string
  game: string
  gameValue: string
  label: string
  description: ReactNode
  options: { value: string; label: string }[]
}) {
  const scoped = useScopedKey(settingKey, game, gameValue)
  return (
    <SettingRow
      label={label}
      description={description}
      {...(scoped.own === undefined ? {} : { mark: <ScopeDot /> })}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
        <PrefSelect
          value={scoped.value}
          options={options}
          label={label}
          onChange={(v) => saveSetting(() => scoped.save(v))}
        />
        {scoped.profileScope && scoped.own !== undefined ? (
          <ScopeReset
            gameLabel={options.find((o) => o.value === scoped.gameValue)?.label ?? scoped.gameValue}
            onReset={() => saveSetting(scoped.reset)}
          />
        ) : null}
      </Box>
    </SettingRow>
  )
}
