import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { PickFolder } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import { SetDownloadFolder } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefByKey, PrefKeys } from '../PrefRow.tsx'
import { persist } from '../persist.ts'
import { SettingsSection } from '../SettingsSection.tsx'

export function DataPrefs() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const fail = t`Couldn't save that setting`
  return (
    <>
      <Box sx={{ fontSize: 14, fontWeight: 600 }}>{t`Stardew Valley`}</Box>
      <SettingsSection title={t`Play backups`} prefKeys={['backupBeforePlay', 'launchBackupsKept']}>
        <PrefKeys keys={['backupBeforePlay', 'launchBackupsKept']} />
      </SettingsSection>
      <SettingsSection
        title={t`Logs`}
        prefKeys={[
          'runsKept',
          'consoleLogCap',
          'consoleLevel',
          'consoleTimestamps',
          'consoleFollow',
        ]}
      >
        <PrefKeys
          keys={['runsKept', 'consoleLogCap', 'consoleLevel', 'consoleTimestamps', 'consoleFollow']}
        />
      </SettingsSection>
      <SettingsSection
        title={t`Store`}
        prefKeys={[
          'keepDownloadArchives',
          'downloadFolder',
          'driftChecks',
          'storeRetentionDays',
          'trashRetentionDays',
          'historyEventsKept',
        ]}
      >
        <PrefByKey prefKey="keepDownloadArchives" />
        <PrefByKey
          prefKey="downloadFolder"
          extra={
            <Button
              variant="outlined"
              onClick={() => {
                persist(
                  async () => {
                    const dir = await PickFolder(t`Download folder`)
                    if (dir) {
                      await SetDownloadFolder(dir)
                    }
                  },
                  push,
                  fail,
                )
              }}
              sx={{ whiteSpace: 'nowrap' }}
            >
              {t`Choose…`}
            </Button>
          }
        />
        <PrefKeys
          keys={['driftChecks', 'storeRetentionDays', 'trashRetentionDays', 'historyEventsKept']}
        />
      </SettingsSection>
    </>
  )
}
