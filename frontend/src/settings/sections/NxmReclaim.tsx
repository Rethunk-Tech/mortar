import { useLingui } from '@lingui/react/macro'
import { useEffect, useState } from 'react'
import type { Takeover } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nxmsvc/models.ts'
import {
  TakeBack,
  Taken,
  Yield,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/nxmsvc/service.ts'
import { sourceLabel } from '../../brand/sources/sourceLabel.ts'
import { listNames } from '../../i18n/list.ts'
import { ConfirmationDialog } from '../../shell/dialog/ConfirmationDialog.tsx'
import { toastError } from '../../toasts/report.ts'
import { useSettings } from '../store.ts'

// Asks to take mod links back when another app (Vortex and r2modman re-register themselves whenever they start)
// took them while Mortar is set to handle them; checked at start and whenever the window regains focus.
export function NxmReclaim() {
  const { t } = useLingui()
  const nexus = useSettings((s) => s.nxmHandled)
  const thunderstore = useSettings(
    (s) =>
      s.thunderstoreHandleLinks === true ||
      s.curseforgeHandleLinks === true ||
      s.vsmoddbHandleLinks === true,
  )
  const [taken, setTaken] = useState<Takeover[]>([])
  useEffect(() => {
    if (!(nexus || thunderstore)) {
      return
    }
    const check = () => {
      Taken()
        .then((list) => setTaken(list ?? []))
        .catch(() => undefined)
    }
    check()
    globalThis.addEventListener('focus', check)
    return () => globalThis.removeEventListener('focus', check)
  }, [nexus, thunderstore])
  const fail = (e: unknown) => toastError(t`Could not change how mod links open`, e)
  const lines = taken.map(({ source, name }) => {
    if (source === 'nexus') {
      return t`${name} now opens ${{ source: sourceLabel('nexus') }} "Mod Manager Download" links.`
    }
    if (source === 'curseforge') {
      return t`${name} now opens ${{ source: sourceLabel('curseforge') }} "Install" links.`
    }
    return source === 'vsmoddb'
      ? t`${name} now opens ${{ source: sourceLabel('vsmoddb') }} "1-click install" links.`
      : t`${name} now opens ${{ source: sourceLabel('thunderstore') }} "Install with Mod Manager" links.`
  })
  const names = listNames([...new Set(taken.map((x) => x.name))])
  return (
    <ConfirmationDialog
      open={taken.length > 0}
      title={t`${names} took over mod links`}
      body={[...lines, t`Take them back for Mortar?`].join(' ')}
      cancelLabel={t`Leave them`}
      confirmLabel={t`Take back`}
      onCancel={() => {
        const sources = taken.map((x) => x.source)
        setTaken([])
        Yield(sources).catch(fail)
      }}
      onConfirm={() => {
        setTaken([])
        TakeBack().catch(fail)
      }}
    />
  )
}
