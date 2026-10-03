import { useLingui } from '@lingui/react/macro'
import {
  SetBackupBeforePlay,
  SetConsoleLogCap,
  SetHistoryEventsKept,
  SetKeepDownloadArchives,
  SetLaunchBackupsKept,
  SetRunsKept,
  SetStoreRetentionDays,
  SetTrashRetentionDays,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefNumber, PrefSelect, PrefSwitch } from '../PrefControls.tsx'
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
