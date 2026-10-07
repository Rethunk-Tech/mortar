import { useLingui } from '@lingui/react/macro'
import type { Bundle } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/bundles/models.ts'
import { List } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/bundles/service.ts'
import { useLoaded } from '../shell/useLoaded.ts'
import { reportError } from '../toasts/report.ts'

const NONE: Bundle[] = []

// The game's bundles; `open` false skips the read (a closed dialog).
export function useBundles(game: string, open = true) {
  const { t } = useLingui()
  const { data, setData, loading } = useLoaded(
    open ? () => List(game).then((listed) => listed ?? NONE) : null,
    [game, open],
    NONE,
    reportError(t`Could not read bundles`),
  )
  return { bundles: data, setBundles: setData, loading }
}
