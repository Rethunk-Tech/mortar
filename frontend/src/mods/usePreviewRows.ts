import { useCallback, useEffect, useRef, useState } from 'react'
import type {
  GameModPreview,
  GameModsPreview,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { type InlineError, inlineError } from '../toasts/report.ts'

interface Rows {
  mods: GameModPreview[]
  loading: boolean
  error: InlineError | null
}

// Reads a folder preview each time the dialog opens; fetchRows must be stable between renders.
export function usePreviewRows(open: boolean, fetchRows: () => Promise<GameModsPreview>) {
  const [rows, setRows] = useState<Rows>({ mods: [], loading: false, error: null })
  const gen = useRef(0)
  const reload = useCallback(() => {
    gen.current += 1
    const token = gen.current
    setRows((s) => ({ ...s, loading: true, error: null }))
    fetchRows()
      .then(
        (p) =>
          token === gen.current && setRows({ mods: p.mods ?? [], loading: false, error: null }),
      )
      .catch(
        (e: unknown) =>
          token === gen.current && setRows({ mods: [], loading: false, error: inlineError(e) }),
      )
  }, [fetchRows])
  useEffect(() => {
    if (open) {
      reload()
    }
    return () => {
      gen.current += 1
    }
  }, [open, reload])
  return {
    ...rows,
    reload,
    drop: (folder: string) =>
      setRows((s) => ({ ...s, mods: s.mods.filter((m) => m.folder !== folder) })),
  }
}
