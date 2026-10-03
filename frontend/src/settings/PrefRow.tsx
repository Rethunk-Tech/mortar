import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import type { ReactNode } from 'react'
import type { PrefSpec } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import { SetByKey } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { useToasts } from '../toasts/store.ts'
import {
  PrefCards,
  PrefNumber,
  PrefSegmented,
  PrefSelect,
  PrefSwitch,
  PrefText,
} from './PrefControls.tsx'
import { persist } from './persist.ts'
import { choiceStyle, prefControl } from './prefControl.ts'
import { prefCopy } from './prefCopy.ts'
import { specByKey, usePrefSpecs } from './prefSpecs.ts'
import { prefAsBool, prefAsNumber, prefAsString, prefRaw, specGameArg } from './prefValue.ts'
import { SettingRow } from './SettingsSection.tsx'
import { useSettings } from './store.ts'

// Game-scope prefs render only where a game is passed, so global pages never write to a guessed game.
export function PrefRow({
  spec,
  extra,
  game,
}: {
  spec: PrefSpec
  extra?: ReactNode
  game?: string
}) {
  const { t, i18n } = useLingui()
  const push = useToasts((s) => s.push)
  const fail = t`Couldn't save that setting`
  const settings = useSettings()
  const copy = prefCopy(i18n, spec.key)
  if (spec.scope === 'game' && !game) {
    return null
  }
  const raw = prefRaw(settings, spec, game)
  const gameArg = specGameArg(spec, game)
  const save = (value: string) => persist(() => SetByKey(spec.key, value, gameArg), push, fail)
  const kind = prefControl(spec.type)
  const options = copy.options ?? (spec.values ?? []).map((value) => ({ value, label: value }))
  let selectValue = prefAsString(raw, spec)
  if (options.length > 0 && !options.some((o) => o.value === selectValue)) {
    selectValue = options[0]?.value ?? selectValue
  }
  const style = options.length > 0 ? choiceStyle(options) : 'select'
  let control: ReactNode
  if (kind === 'switch') {
    control = <PrefSwitch checked={prefAsBool(raw, spec)} onChange={(on) => save(String(on))} />
  } else if (style === 'cards') {
    control = <PrefCards value={selectValue} onChange={save} options={options} label={copy.label} />
  } else if (style === 'segmented' && options.length > 0) {
    control = (
      <PrefSegmented value={selectValue} onChange={save} options={options} label={copy.label} />
    )
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
        {...(copy.placeholder === undefined ? {} : { placeholder: copy.placeholder })}
        onCommit={(v) => SetByKey(spec.key, v, gameArg)}
      />
    )
  }
  return (
    <SettingRow label={copy.label} description={copy.description} block={style === 'cards'}>
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

export function PrefByKey({
  prefKey,
  extra,
  game,
}: {
  prefKey: string
  extra?: ReactNode
  game?: string
}) {
  const spec = specByKey(usePrefSpecs(), prefKey)
  if (!spec) {
    return null
  }
  return <PrefRow spec={spec} extra={extra} {...(game ? { game } : {})} />
}

export function PrefKeys({ keys, game }: { keys: string[]; game?: string }) {
  return (
    <>
      {keys.map((key) => (
        <PrefByKey key={key} prefKey={key} {...(game ? { game } : {})} />
      ))}
    </>
  )
}
