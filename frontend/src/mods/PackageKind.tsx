import { useLingui } from '@lingui/react/macro'

// What a BepInEx package ships, read from its DLLs (ModFact.kind).
export function PackageKind({ kind }: { kind: string }) {
  const { t } = useLingui()
  const names: Record<string, string> = {
    plugin: t`Plugin`,
    patcher: t`Patcher`,
    both: t`Plugin and patcher`,
    library: t`Library`,
    assets: t`Files only`,
  }
  return names[kind] ?? '—'
}
