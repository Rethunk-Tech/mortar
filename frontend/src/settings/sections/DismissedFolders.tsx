import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { UndismissGameModsFolders } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useCurrentGame } from '../../nav/currentGame.ts'
import { reportError } from '../../toasts/report.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

const NONE: string[] = []

// Folders in the game's Mods folder the user chose not to move into a profile, each with a way to offer it again.
export function DismissedFolders() {
  const { t } = useLingui()
  const game = useCurrentGame()
  const folders = useSettings((s) => s.dismissed?.[`gameMods:${game}`] ?? NONE)
  if (folders.length === 0) {
    return null
  }
  return (
    <SettingsSection title={t`Dismissed folders`}>
      {folders.map((folder) => (
        <SettingRow
          key={folder}
          label={folder}
          description={t`Not offered for moving into a profile.`}
        >
          <Button
            variant="outlined"
            onClick={() =>
              UndismissGameModsFolders(game, [folder]).catch(
                reportError(t`Could not offer the folder again`),
              )
            }
          >
            {t`Offer again`}
          </Button>
        </SettingRow>
      ))}
    </SettingsSection>
  )
}
