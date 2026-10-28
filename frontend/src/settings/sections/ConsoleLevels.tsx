import { useLingui } from '@lingui/react/macro'
import { Box, Checkbox, FormControlLabel } from '@mui/material'
import { LEVELS, levelsFromFloor } from '../../console/filter.ts'
import { useLevelNames } from '../../console/logHooks.ts'
import { gamePrefs } from '../gamePrefs.ts'
import { saveSetting } from '../persist.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useScopedKey } from '../scopedKey.ts'
import { useSettings } from '../store.ts'

/** The levels the game's Console shows when it opens, each ticked on or off; one stays on. */
export function ConsoleLevels({ game }: { game: string }) {
  const { t } = useLingui()
  const names = useLevelNames()
  const scoped = useScopedKey(
    'consoleLevel',
    game,
    useSettings((s) => gamePrefs(s, game).consoleLevel),
  )
  const shown = levelsFromFloor(scoped.value)
  const save = (next: typeof shown) => saveSetting(() => scoped.save(next.join(',')))
  return (
    <SettingRow label={t`Levels shown`} description={t`The Console opens showing these levels`}>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', justifyContent: 'flex-end' }}>
        {LEVELS.map((level) => {
          const on = shown.includes(level)
          return (
            <FormControlLabel
              key={level}
              label={names[level]}
              control={
                <Checkbox
                  size="small"
                  checked={on}
                  disabled={on && shown.length === 1}
                  onChange={() =>
                    save(
                      on
                        ? shown.filter((l) => l !== level)
                        : LEVELS.filter((l) => l === level || shown.includes(l)),
                    )
                  }
                />
              }
            />
          )
        })}
      </Box>
    </SettingRow>
  )
}
