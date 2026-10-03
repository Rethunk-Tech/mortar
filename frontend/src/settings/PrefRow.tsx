import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import type { ReactNode } from 'react'
import type { PrefSpec } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import { SetByKey } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useToasts } from '../toasts/store.ts'
import { PrefNumber, PrefSelect, PrefSwitch, PrefText } from './PrefControls.tsx'
import { persist } from './persist.ts'
import { prefControl } from './prefControl.ts'
import { prefCopy } from './prefCopy.ts'
import { specByKey, usePrefSpecs } from './prefSpecs.ts'
import {
  GAME_STARDEW,
  prefAsBool,
  prefAsNumber,
  prefAsString,
  prefRaw,
  specGameArg,
} from './prefValue.ts'
import { SettingRow } from './SettingsSection.tsx'
import { useSettings } from './store.ts'

export function PrefRow({ spec, extra }: { spec: PrefSpec; extra?: ReactNode }) {
  const { t } = useLingui()
  const push = useToasts((s) => s.push)
  const fail = t`Couldn't save that setting`
  const settings = useSettings()
  const game = settings.lastGame || GAME_STARDEW
  const copy = prefCopy(t, spec.key)
  const raw = prefRaw(settings, spec, game)
  const gameArg = specGameArg(spec, game)
  const save = (value: string) => persist(() => SetByKey(spec.key, value, gameArg), push, fail)
  const kind = prefControl(spec.type)
  const options = copy.options ?? (spec.values ?? []).map((value) => ({ value, label: value }))
  let selectValue = prefAsString(raw, spec)
  if (options.length > 0 && !options.some((o) => o.value === selectValue)) {
    selectValue = options[0]?.value ?? selectValue
  }
  let control: ReactNode
  if (kind === 'switch') {
    control = <PrefSwitch checked={prefAsBool(raw, spec)} onChange={(on) => save(String(on))} />
  } else if (kind === 'select' || options.length > 0) {
    control = <PrefSelect value={selectValue} onChange={save} options={options} />
  } else if (kind === 'number') {
    control = (
      <PrefNumber
        value={prefAsNumber(raw, spec)}
        min={spec.min ?? 0}
        max={spec.max || Number.MAX_SAFE_INTEGER}
        onCommit={(n) => SetByKey(spec.key, String(n), gameArg)}
      />
    )
  } else {
    control = (
      <PrefText
        value={prefAsString(raw, spec)}
        placeholder={copy.placeholder}
        onCommit={(v) => SetByKey(spec.key, v, gameArg)}
      />
    )
  }
  return (
    <SettingRow label={copy.label} description={copy.description}>
      {extra ? (
        <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
          {control}
          {extra}
        </Box>
      ) : (
        control
      )}
    </SettingRow>
  )
}

export function PrefByKey({ prefKey, extra }: { prefKey: string; extra?: ReactNode }) {
  const spec = specByKey(usePrefSpecs(), prefKey)
  if (!spec) {
    return null
  }
  return <PrefRow spec={spec} extra={extra} />
}

export function PrefKeys({ keys }: { keys: string[] }) {
  return (
    <>
      {keys.map((key) => (
        <PrefByKey key={key} prefKey={key} />
      ))}
    </>
  )
}
