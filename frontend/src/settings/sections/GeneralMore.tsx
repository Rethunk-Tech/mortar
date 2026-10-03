import { useLingui } from '@lingui/react/macro'
import {
  SetBackgroundBadgeChecks,
  SetConfirmRemovals,
  SetCosmeticConflicts,
  SetDates,
  SetDefaultLaunchMethod,
  SetDefaultModsView,
  SetDensity,
  SetEnableRequirements,
  SetGridCardSize,
  SetLanAutoAcceptSameAccount,
  SetLanName,
  SetListGroupBy,
  SetListSort,
  SetMissingRequirements,
  SetNotifyDownloadFailed,
  SetNotifyDownloadFinished,
  SetNotifyRunCrashed,
  SetOnPlay,
  SetProfileHero,
  SetReduceMotion,
  SetReuseFomodChoices,
  SetShowAuthorOnCards,
  SetShowSmapiConsole,
  SetStartScreen,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useToasts } from '../../toasts/store.ts'
import { PrefSelect, PrefSwitch, PrefText } from '../PrefControls.tsx'
import { persist } from '../persist.ts'
import { SettingRow, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

export function LanIdentity() {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const fail = t`Couldn't save that setting`
  return (
    <>
      <SettingRow
        label={t`Device name`}
        description={t`How this Mortar appears to nearby installations`}
      >
        <PrefText
          value={useSettings((s) => s.lanName)}
          placeholder={t`This computer`}
          onCommit={(v) => SetLanName(v)}
        />
      </SettingRow>
      <SettingRow
        label={t`Auto-accept from this Nexus account`}
        description={t`Take LAN shares from machines signed in to the same Nexus account`}
      >
        <PrefSwitch
          checked={useSettings((s) => s.lanAutoAcceptSameAccount)}
          onChange={(on) => persist(() => SetLanAutoAcceptSameAccount(on), push, fail)}
        />
      </SettingRow>
    </>
  )
}

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
      <SettingRow
        label={t`Default launch`}
        description={t`How Play starts the game. Direct skips Steam's overlay and playtime.`}
      >
        <PrefSelect
          value={useSettings((s) => s.defaultLaunchMethod) || 'steam'}
          onChange={(v) => persist(() => SetDefaultLaunchMethod(v), push, fail)}
          options={[
            { value: 'steam', label: t`Steam` },
            { value: 'direct', label: t`Direct` },
          ]}
        />
      </SettingRow>
      <SettingRow
        label={t`SMAPI console window`}
        description={t`Show SMAPI's own window on a direct launch`}
      >
        <PrefSwitch
          checked={useSettings((s) => s.showSmapiConsole) !== false}
          onChange={(on) => persist(() => SetShowSmapiConsole(on), push, fail)}
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
      <SettingRow label={t`Grid card size`}>
        <PrefSelect
          value={useSettings((s) => s.gridCardSize) || 'medium'}
          onChange={(v) => persist(() => SetGridCardSize(v), push, fail)}
          options={[
            { value: 'small', label: t`Small` },
            { value: 'medium', label: t`Medium` },
            { value: 'large', label: t`Large` },
          ]}
        />
      </SettingRow>
      <SettingRow label={t`Author on cards`} description={t`Show the author line on grid cards`}>
        <PrefSwitch
          checked={useSettings((s) => s.showAuthorOnCards) !== false}
          onChange={(on) => persist(() => SetShowAuthorOnCards(on), push, fail)}
        />
      </SettingRow>
      <SettingRow
        label={t`Auto-enable requirements`}
        description={t`When you switch a mod on, also enable its required mods already in the profile`}
      >
        <PrefSelect
          value={useSettings((s) => s.enableRequirements) || 'always'}
          onChange={(v) => persist(() => SetEnableRequirements(v), push, fail)}
          options={[
            { value: 'always', label: t`Always` },
            { value: 'ask', label: t`Ask` },
            { value: 'never', label: t`Never` },
          ]}
        />
      </SettingRow>
      <SettingRow
        label={t`Missing requirements on install`}
        description={t`What to do when an installed mod still needs other mods`}
      >
        <PrefSelect
          value={useSettings((s) => s.missingRequirements) || 'ask'}
          onChange={(v) => persist(() => SetMissingRequirements(v), push, fail)}
          options={[
            { value: 'ask', label: t`Ask` },
            { value: 'autodownload', label: t`Download them` },
            { value: 'never', label: t`Never` },
          ]}
        />
      </SettingRow>
      <SettingRow
        label={t`Reuse FOMOD choices`}
        description={t`Skip the installer wizard when saved choices still fit`}
      >
        <PrefSwitch
          checked={useSettings((s) => s.reuseFomodChoices) !== false}
          onChange={(on) => persist(() => SetReuseFomodChoices(on), push, fail)}
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
        <SettingRow label={t`Density`} description={t`Spacing of buttons and lists`}>
          <PrefSelect
            value={useSettings((s) => s.density) || 'comfortable'}
            onChange={(v) => persist(() => SetDensity(v), push, fail)}
            options={[
              { value: 'comfortable', label: t`Comfortable` },
              { value: 'compact', label: t`Compact` },
            ]}
          />
        </SettingRow>
        <SettingRow
          label={t`Reduce motion`}
          description={t`Shorter animations. Honour the OS unless you override it.`}
        >
          <PrefSelect
            value={useSettings((s) => s.reduceMotion) || 'system'}
            onChange={(v) => persist(() => SetReduceMotion(v), push, fail)}
            options={[
              { value: 'system', label: t`Honour OS` },
              { value: 'always', label: t`Always` },
              { value: 'never', label: t`Never` },
            ]}
          />
        </SettingRow>
        <SettingRow label={t`Profile hero`} description={t`The banner at the top of a profile`}>
          <PrefSelect
            value={useSettings((s) => s.profileHero) || 'full'}
            onChange={(v) => persist(() => SetProfileHero(v), push, fail)}
            options={[
              { value: 'full', label: t`Full` },
              { value: 'compact', label: t`Compact` },
              { value: 'hidden', label: t`Hidden` },
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
