import { useLingui } from '@lingui/react/macro'
import { EyeOff, SlidersHorizontal, Wrench } from 'lucide-react'
import { RememberSettingChoice } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts'
import { SetConfigValue } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import type { Problem } from '../lookup.ts'
import { useMods } from '../store.ts'
import { openTarget } from '../storeView.ts'
import { useLocked } from '../useLocked.ts'
import { FixAction } from './fixActions.tsx'
import { RestoreButton } from './RestoreButton.tsx'

export function SettingFix({
  problem,
  dismissedToken,
}: {
  problem: Extract<Problem, { kind: 'setting' }>
  dismissedToken?: string | undefined
}) {
  const { t } = useLingui()
  const { setting } = problem
  const loadProblems = useMods((s) => s.loadProblems)
  const dismissSetting = useMods((s) => s.dismissSetting)
  const locked = useLocked()
  const values = setting.suggested ?? []
  const apply = async (value: string) => {
    const at = openTarget()
    if (!at) {
      return
    }
    try {
      await SetConfigValue(at.game, at.id, setting.key, setting.id, setting.field, value)
      await RememberSettingChoice(at.game, at.id, setting.id, setting.field, value)
      await loadProblems()
    } catch (error) {
      reportUnexpected(error)
    }
  }
  if (values.length === 0) {
    return null
  }
  const first = values[0] ?? ''
  const label = (value: string) => (value === '' ? t`Set to automatic` : t`Set to ${value}`)
  return (
    <>
      {dismissedToken === undefined ? (
        <FixAction
          icon={<Wrench size={16} aria-hidden={true} />}
          label={label(first)}
          weight="main"
          locked={locked}
          onClick={() => apply(first)}
        />
      ) : (
        <RestoreButton token={dismissedToken} />
      )}
      {values.length > 1 ? (
        <FixAction
          icon={<SlidersHorizontal size={16} aria-hidden={true} />}
          label={t`Other values`}
          weight="other"
          locked={locked}
          choices={values
            .slice(1)
            .map((value) => ({ label: label(value), onClick: () => apply(value) }))}
        />
      ) : null}
      {dismissedToken === undefined ? (
        <FixAction
          icon={<EyeOff size={16} aria-hidden={true} />}
          label={t`Dismiss`}
          weight="quiet"
          onClick={() => dismissSetting(setting).catch(reportUnexpected)}
        />
      ) : null}
    </>
  )
}
