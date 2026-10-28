import { useLingui } from '@lingui/react/macro'
import { ToggleButton, ToggleButtonGroup } from '@mui/material'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { useSettingsScope } from './settingsScope.ts'
import { useProfileScope } from './useProfileScope.ts'

/** Switches a game's settings between every profile's values and the open profile's own. */
export function ScopeSwitch({ gameName }: { gameName: string }) {
  const { t } = useLingui()
  const profileName = useProfiles((s) => openProfileOf(s)?.name ?? '')
  const profile = useProfileScope()
  if (profileName === '') {
    return null
  }
  return (
    <ToggleButtonGroup
      exclusive={true}
      size="small"
      value={profile ? 'profile' : 'game'}
      onChange={(_, next: string | null) =>
        next && useSettingsScope.setState({ profile: next === 'profile' })
      }
      aria-label={t`Settings for`}
    >
      <ToggleButton value="game">{t`All ${{ game: gameName }} profiles`}</ToggleButton>
      <ToggleButton value="profile">{t`This profile: ${{ profile: profileName }}`}</ToggleButton>
    </ToggleButtonGroup>
  )
}
