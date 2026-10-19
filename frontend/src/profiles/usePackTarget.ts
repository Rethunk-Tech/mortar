import { useCallback, useEffect, useState } from 'react'
import type {
  Extra,
  Source,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/models.ts'
import {
  Extras,
  Import,
  ImportAs,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  List as ListProfiles,
  RemoveEntries,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { reportUnexpected } from '../toasts/report.ts'

export interface PackTargetState {
  existing: Profile[]
  // '' imports into a new profile named `name`; otherwise the id of the profile to update.
  target: string
  name: string
  extras: Extra[]
  dropExtras: boolean
  replaceFiles: boolean
  choose: (id: string) => void
  setName: (name: string) => void
  setDropExtras: (drop: boolean) => void
  setReplaceFiles: (replace: boolean) => void
  reset: () => void
  importPack: () => ReturnType<typeof Import>
}

// The state of where an open import dialog puts its pack, and the import itself. `source` reads the pack the player
// handed over.
export function usePackTarget(open: boolean, game: string, source: () => Source): PackTargetState {
  const [existing, setExisting] = useState<Profile[]>([])
  const [target, setTarget] = useState('')
  const [name, setName] = useState('')
  const [extras, setExtras] = useState<Extra[]>([])
  const [dropExtras, setDropExtras] = useState(true)
  const [replaceFiles, setReplaceFiles] = useState(false)
  useEffect(() => {
    if (open) {
      ListProfiles(game)
        .then((all) => setExisting((all ?? []).filter((p) => !p.error)))
        .catch(reportUnexpected)
    }
  }, [open, game])
  const choose = (id: string) => {
    setTarget(id)
    setExtras([])
    if (id) {
      Extras(source(), game, id)
        .then((found) => setExtras(found ?? []))
        .catch(reportUnexpected)
    }
  }
  const reset = useCallback(() => {
    setTarget('')
    setName('')
    setExtras([])
    setReplaceFiles(false)
  }, [])
  const importPack = () => {
    if (!target) {
      return ImportAs(source(), game, name.trim())
    }
    return Import({ ...source(), replace: replaceFiles }, game, target).then(async (result) => {
      if (dropExtras && extras.length > 0) {
        await RemoveEntries(
          game,
          target,
          extras.map((e) => e.key),
        )
      }
      return result
    })
  }
  return {
    existing,
    target,
    name,
    extras,
    dropExtras,
    replaceFiles,
    choose,
    setName,
    setDropExtras,
    setReplaceFiles,
    reset,
    importPack,
  }
}
