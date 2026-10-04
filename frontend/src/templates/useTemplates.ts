import { useLingui } from '@lingui/react/macro'
import { useCallback, useEffect, useState } from 'react'
import type { Template } from '../../bindings/github.com/Rethunk-AI/mortar/internal/templates/models.ts'
import { Templates } from '../../bindings/github.com/Rethunk-AI/mortar/internal/templates/service.ts'
import { reportError } from '../toasts/report.ts'

/** The game's templates, read when `active` turns on and again on `reload`. */
export function useTemplates(game: string, active: boolean) {
  const { t } = useLingui()
  const [templates, setTemplates] = useState<Template[]>([])
  const reload = useCallback(
    () =>
      game
        ? Templates(game)
            .then((found) => setTemplates(found ?? []))
            .catch(reportError(t`Could not read the templates`))
        : Promise.resolve(),
    [game, t],
  )
  useEffect(() => {
    if (active) {
      reload().catch(() => undefined)
    }
  }, [active, reload])
  return { templates, reload }
}
