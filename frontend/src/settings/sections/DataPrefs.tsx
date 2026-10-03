import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { PickFolder } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/picker/service.ts'
import {
  SetBackupBeforePlay,
  SetConsoleFollow,
  SetConsoleLevel,
  SetConsoleLogCap,
  SetConsoleTimestamps,
  SetDownloadFolder,
  SetDriftChecks,
  SetHistoryEventsKept,
  SetKeepDownloadArchives,
  SetLaunchBackupsKept,
  SetRunsKept,
  SetStoreRetentionDays,
  SetTrashRetentionDays,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefNumber, PrefSelect, PrefSwitch, PrefText } from '../PrefControls.tsx'
import { persist } from '../persist.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

const defaultLaunchBackups = 5
const defaultRunsKept = 20
const defaultConsoleLogCap = 20_000
const defaultTrashDays = 30
const defaultHistoryKept = 200

export function DataPrefs() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const fail = t`Couldn't save that setting`
  return (
    <>
      <SettingsSection title={t`Play backups`}>
        <SettingRow
          label={t`Backup before Play`}
          description={t`When Mortar zips Saves before launching`}
        >
          <PrefSelect
            value={useSettings((s) => s.backupBeforePlay) || 'changed'}
            onChange={(v) => persist(() => SetBackupBeforePlay(v), push, fail)}
            options={[
              { value: 'changed', label: t`When mods changed` },
              { value: 'always', label: t`Every Play` },
              { value: 'never', label: t`Never` },
            ]}
          />
        </SettingRow>
        <SettingRow label={t`Launch backups kept`}>
          <PrefNumber
            value={useSettings((s) => s.launchBackupsKept) || defaultLaunchBackups}
            min={1}
            max={50}
            onCommit={(n) => SetLaunchBackupsKept(n)}
          />
        </SettingRow>
      </SettingsSection>
      <SettingsSection title={t`Logs`}>
        <SettingRow label={t`Run logs kept`} description={t`Stored SMAPI logs per profile`}>
          <PrefNumber
            value={useSettings((s) => s.runsKept) || defaultRunsKept}
            min={1}
            max={100}
            onCommit={(n) => SetRunsKept(n)}
          />
        </SettingRow>
        <SettingRow label={t`Console log cap`} description={t`Newest lines kept in the Console`}>
          <PrefNumber
            value={useSettings((s) => s.consoleLogCap) || defaultConsoleLogCap}
            min={1000}
            max={100_000}
            onCommit={(n) => SetConsoleLogCap(n)}
          />
        </SettingRow>
        <SettingRow
          label={t`Console level`}
          description={t`Live log starts at this level and above`}
        >
          <PrefSelect
            value={useSettings((s) => s.consoleLevel) || 'info'}
            onChange={(v) => persist(() => SetConsoleLevel(v), push, fail)}
            options={[
              { value: 'trace', label: t`Trace` },
              { value: 'debug', label: t`Debug` },
              { value: 'info', label: t`Info` },
              { value: 'warn', label: t`Warn` },
              { value: 'error', label: t`Error` },
            ]}
          />
        </SettingRow>
        <SettingRow label={t`Console timestamps`}>
          <PrefSwitch
            checked={useSettings((s) => s.consoleTimestamps) !== false}
            onChange={(on) => persist(() => SetConsoleTimestamps(on), push, fail)}
          />
        </SettingRow>
        <SettingRow label={t`Follow live log`}>
          <PrefSwitch
            checked={useSettings((s) => s.consoleFollow) !== false}
            onChange={(on) => persist(() => SetConsoleFollow(on), push, fail)}
          />
        </SettingRow>
      </SettingsSection>
      <SettingsSection title={t`Store`}>
        <SettingRow
          label={t`Keep downloaded archives`}
          description={t`Leave the zip after it is installed into the store`}
        >
          <PrefSwitch
            checked={useSettings((s) => s.keepDownloadArchives)}
            onChange={(on) => persist(() => SetKeepDownloadArchives(on), push, fail)}
          />
        </SettingRow>
        <SettingRow
          label={t`Download folder`}
          description={t`Archives land here instead of Mortar's downloads folder. Empty uses the default.`}
        >
          <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
            <PrefText
              value={useSettings((s) => s.downloadFolder)}
              onCommit={(v) => SetDownloadFolder(v)}
            />
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
          </Box>
        </SettingRow>
        <SettingRow
          label={t`Modified outside Mortar`}
          description={t`Scan the mods folder for changes Mortar did not make`}
        >
          <PrefSwitch
            checked={useSettings((s) => s.driftChecks) !== false}
            onChange={(on) => persist(() => SetDriftChecks(on), push, fail)}
          />
        </SettingRow>
        <SettingRow
          label={t`Unused store items`}
          description={t`Days to keep unused store items. 0 keeps them forever.`}
        >
          <PrefNumber
            value={useSettings((s) => s.storeRetentionDays)}
            min={0}
            max={3650}
            onCommit={(n) => SetStoreRetentionDays(n)}
          />
        </SettingRow>
        <SettingRow
          label={t`Trash retention`}
          description={t`Days a deleted profile stays restorable`}
        >
          <PrefNumber
            value={useSettings((s) => s.trashRetentionDays) || defaultTrashDays}
            min={1}
            max={365}
            onCommit={(n) => SetTrashRetentionDays(n)}
          />
        </SettingRow>
        <SettingRow label={t`History events kept`} description={t`Per profile`}>
          <PrefNumber
            value={useSettings((s) => s.historyEventsKept) || defaultHistoryKept}
            min={20}
            max={2000}
            onCommit={(n) => SetHistoryEventsKept(n)}
          />
        </SettingRow>
      </SettingsSection>
    </>
  )
}
