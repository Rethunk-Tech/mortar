import { useLingui } from '@lingui/react/macro'
import {
  SetBackgroundBadgeChecks,
  SetConfirmRemovals,
  SetCosmeticConflicts,
  SetDates,
  SetDefaultModsView,
  SetListGroupBy,
  SetListSort,
  SetNotifyDownloadFailed,
  SetNotifyDownloadFinished,
  SetNotifyRunCrashed,
  SetOnPlay,
  SetStartScreen,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefSelect, PrefSwitch } from '../PrefControls.tsx'
import { persist } from '../persist.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

export function WindowLaunch() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const fail = t`Couldn't save that setting`
  return (
    <>
      <SettingRow
        label={t`When you press Play`}
        description={t`What Mortar does when the game starts, then restore when it exits.`}
      >
        <PrefSelect
          value={useSettings((s) => s.onPlay) || 'stay'}
          onChange={(v) => persist(() => SetOnPlay(v), push, fail)}
          options={[
            { value: 'stay', label: t`Stay open` },
            { value: 'minimise', label: t`Minimise` },
            { value: 'hide', label: t`Hide to tray` },
          ]}
        />
      </SettingRow>
      <SettingRow
        label={t`Start screen`}
        description={t`Where Mortar opens after the launcher check.`}
      >
        <PrefSelect
          value={useSettings((s) => s.startScreen) || 'last'}
          onChange={(v) => persist(() => SetStartScreen(v), push, fail)}
          options={[
            { value: 'last', label: t`Last opened profile` },
            { value: 'gameselect', label: t`Game Select` },
          ]}
        />
      </SettingRow>
    </>
  )
}

export function ModsPrefs() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const fail = t`Couldn't save that setting`
  return (
    <SettingsSection title={t`Mods`}>
      <SettingRow label={t`Default Mods view`} description={t`Grid or list for new sessions`}>
        <PrefSelect
          value={useSettings((s) => s.defaultModsView) || 'grid'}
          onChange={(v) => persist(() => SetDefaultModsView(v), push, fail)}
          options={[
            { value: 'grid', label: t`Grid` },
            { value: 'list', label: t`List` },
          ]}
        />
      </SettingRow>
      <SettingRow label={t`Default grouping`}>
        <PrefSelect
          value={useSettings((s) => s.listGroupBy) || 'status'}
          onChange={(v) => persist(() => SetListGroupBy(v), push, fail)}
          options={[
            { value: 'none', label: t`None` },
            { value: 'status', label: t`Status` },
            { value: 'category', label: t`Category` },
            { value: 'source', label: t`Source` },
            { value: 'tag', label: t`Tag` },
            { value: 'framework', label: t`Framework` },
            { value: 'author', label: t`Author` },
          ]}
        />
      </SettingRow>
      <SettingRow label={t`Default sort`}>
        <PrefSelect
          value={`${useSettings((s) => s.listSortColumn) || 'name'}:${useSettings((s) => s.listSortDir) || 'asc'}`}
          onChange={(v) => {
            const [column, dir] = v.split(':')
            persist(() => SetListSort(column ?? 'name', dir ?? 'asc'), push, fail)
          }}
          options={[
            { value: 'name:asc', label: t`Name A–Z` },
            { value: 'name:desc', label: t`Name Z–A` },
            { value: 'version:asc', label: t`Version` },
            { value: 'author:asc', label: t`Author` },
          ]}
        />
      </SettingRow>
      <SettingRow
        label={t`Confirm removals`}
        description={t`Ask before removing mods from a profile`}
      >
        <PrefSwitch
          checked={useSettings((s) => s.confirmRemovals) !== false}
          onChange={(on) => persist(() => SetConfirmRemovals(on), push, fail)}
        />
      </SettingRow>
      <SettingRow
        label={t`Harmless conflicts`}
        description={t`Cosmetic overlaps on the Problems tab`}
      >
        <PrefSelect
          value={useSettings((s) => s.cosmeticConflicts) || 'collapsed'}
          onChange={(v) => persist(() => SetCosmeticConflicts(v), push, fail)}
          options={[
            { value: 'collapsed', label: t`Collapsed` },
            { value: 'expanded', label: t`Expanded` },
            { value: 'hidden', label: t`Hidden` },
          ]}
        />
      </SettingRow>
      <SettingRow
        label={t`Badge checks for other profiles`}
        description={t`Check updates and problems in the background for profiles you are not looking at`}
      >
        <PrefSwitch
          checked={useSettings((s) => s.backgroundBadgeChecks) !== false}
          onChange={(on) => persist(() => SetBackgroundBadgeChecks(on), push, fail)}
        />
      </SettingRow>
    </SettingsSection>
  )
}

export function DisplayAndNotices() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const fail = t`Couldn't save that setting`
  return (
    <>
      <SettingsSection title={t`Display`}>
        <SettingRow label={t`Dates`} description={t`How timestamps are shown`}>
          <PrefSelect
            value={useSettings((s) => s.dates) || 'relative'}
            onChange={(v) => persist(() => SetDates(v), push, fail)}
            options={[
              { value: 'relative', label: t`Relative` },
              { value: 'absolute', label: t`Absolute` },
            ]}
          />
        </SettingRow>
      </SettingsSection>
      <SettingsSection title={t`Notifications`}>
        <SettingRow label={t`Downloads finished`}>
          <PrefSwitch
            checked={useSettings((s) => s.notifyDownloadFinished) !== false}
            onChange={(on) => persist(() => SetNotifyDownloadFinished(on), push, fail)}
          />
        </SettingRow>
        <SettingRow label={t`Download failed`}>
          <PrefSwitch
            checked={useSettings((s) => s.notifyDownloadFailed) !== false}
            onChange={(on) => persist(() => SetNotifyDownloadFailed(on), push, fail)}
          />
        </SettingRow>
        <SettingRow label={t`Run crashed`}>
          <PrefSwitch
            checked={useSettings((s) => s.notifyRunCrashed) !== false}
            onChange={(on) => persist(() => SetNotifyRunCrashed(on), push, fail)}
          />
        </SettingRow>
      </SettingsSection>
    </>
  )
}
