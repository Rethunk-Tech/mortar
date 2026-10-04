import { useLingui } from '@lingui/react/macro'
import { formatWhen } from '../../i18n/formatWhen.ts'
import { hoursUntilNext, useLastScheduled } from '../../saves/scheduledBackups.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

const GAME = 'stardew'

// When the scheduled save backup last ran and when the next is due, under the interval setting.
export function ScheduledStatus() {
  const { t } = useLingui()
  const hours = useSettings((s) => s.games?.[GAME]?.saveBackupHours ?? 0)
  const last = useLastScheduled()
  let text = t`No scheduled backup has run yet.`
  if (last > 0) {
    const when = formatWhen(last)
    const next = hoursUntilNext(last, hours)
    if (hours === 0) {
      text = t`Last scheduled backup ${when}`
    } else {
      text =
        next === 0
          ? t`Last scheduled backup ${when} · next one is due`
          : t`Last scheduled backup ${when} · next in ${next} h`
    }
  }
  return (
    <SettingRow label={t`Scheduled backup status`} description={text}>
      {null}
    </SettingRow>
  )
}
