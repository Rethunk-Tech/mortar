import { useEffect } from 'react'
import { SHOW_ALL, visible } from './filter.ts'
import { useShownEntries, useVisible } from './logHooks.ts'
import { useConsole } from './store.ts'

// Scrolls to the entry a search hit named once its run is loaded, widening the filters when they hide it.
export function useJumpToSearchHit() {
  const pendingSeq = useConsole((s) => s.pendingSeq)
  const filters = useConsole((s) => s.filters)
  const entries = useShownEntries()
  const rows = useVisible()
  useEffect(() => {
    if (pendingSeq === 0 || entries.length === 0) {
      return
    }
    const { showAll, jumpTo } = useConsole.getState()
    const hidden = !visible(entries, filters).some((e) => e.seq === pendingSeq)
    if (hidden && filters !== SHOW_ALL) {
      showAll()
      return
    }
    useConsole.setState({ pendingSeq: 0 })
    jumpTo(
      Math.max(
        0,
        rows.findLastIndex((r) => r.seq <= pendingSeq),
      ),
    )
  }, [pendingSeq, entries, filters, rows])
}
