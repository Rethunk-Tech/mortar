import { useLingui } from '@lingui/react/macro'
import { Outcome } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launch/models.ts'

// useOutcomeLabel words how a recorded run ended.
export function useOutcomeLabel() {
  const { t } = useLingui()
  return (o: string) => {
    if (o === Outcome.OutcomeCrashed) {
      return t`Crashed`
    }
    if (o === Outcome.OutcomeFailed) {
      return t`Failed`
    }
    return t`Ran`
  }
}
