import { useLingui } from '@lingui/react/macro'
import { formatWhen } from '../../i18n/formatWhen.ts'
import { useCurrentGame } from '../../nav/currentGame.ts'
import { hoursUntilNext, useLastScheduled } from '../../saves/scheduledBackups.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

// When the scheduled save backup last ran and when the next is due, under the interval setting.
export function ScheduledStatus() {
  const { t } = useLingui()
  const game = useCurrentGame()
  const hours = useSettings((s) => s.games?.[game]?.saveBackupHours ?? 0)
  const last = useLastScheduled()
  let text = t`No scheduled backup has run yet.`
  if (last > 0) {
    const when = formatWhen(last)
    const next = hoursUntilNext(last, hours)
    if (hours === 0) {
      text = t`Last scheduled backup ${when}`
    } else {
      text = [
        t`Last scheduled backup ${when}`,
        next === 0 ? t`next one is due` : t`next in ${next} h`,
      ].join(' · ')
    }
  }
  return (
    <SettingRow label={t`Scheduled backup status`} description={text}>
      {null}
    </SettingRow>
  )
}
