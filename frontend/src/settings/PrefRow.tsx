import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import type { ReactNode } from 'react'
import type { PrefSpec } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { space } from '../theme/density.ts'
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
function PrefRow({
  spec,
  extra,
  game,
  disabledReason = '',
}: {
  spec: PrefSpec
  extra?: ReactNode
  game?: string
  disabledReason?: string
}) {
  const { t, i18n } = useLingui()
  const fail = t`Could not save that setting`
  const settings = useSettings()
  const copy = prefCopy(i18n, spec.key)
  if (spec.scope === 'game' && !game) {
    return null
  }
  const raw = prefRaw(settings, spec, game)
  const gameArg = specGameArg(spec, game)
  const save = (value: string) => persist(() => SetByKey(spec.key, value, gameArg), fail)
  const kind = prefControl(spec.type)
  const options = copy.options ?? (spec.values ?? []).map((value) => ({ value, label: value }))
  let selectValue = prefAsString(raw, spec)
  if (options.length > 0 && !options.some((o) => o.value === selectValue)) {
    selectValue = options[0]?.value ?? selectValue
  }
  const style = options.length > 0 ? choiceStyle(options) : 'select'
  let control: ReactNode
  if (kind === 'switch') {
    control = (
      <PrefSwitch
        checked={prefAsBool(raw, spec)}
        onChange={(on) => save(String(on))}
        label={copy.label}
      />
    )
  } else if (style === 'cards') {
    control = <PrefCards value={selectValue} onChange={save} options={options} label={copy.label} />
  } else if (style === 'segmented' && options.length > 0) {
    control = (
      <PrefSegmented value={selectValue} onChange={save} options={options} label={copy.label} />
    )
  } else if (kind === 'select' || options.length > 0) {
    control = (
      <PrefSelect value={selectValue} onChange={save} options={options} label={copy.label} />
    )
  } else if (kind === 'number') {
    control = (
      <PrefNumber
        value={prefAsNumber(raw, spec)}
        min={spec.min ?? 0}
        max={spec.max || Number.MAX_SAFE_INTEGER}
        onCommit={(n) => SetByKey(spec.key, String(n), gameArg)}
        label={copy.label}
        disabled={disabledReason !== ''}
      />
    )
  } else {
    control = (
      <PrefText
        value={prefAsString(raw, spec)}
        {...(copy.placeholder === undefined ? {} : { placeholder: copy.placeholder })}
        onCommit={(v) => SetByKey(spec.key, v, gameArg)}
        label={copy.label}
      />
    )
  }
  return (
    <SettingRow label={copy.label} description={copy.description} block={style === 'cards'}>
      <DisabledReason title={disabledReason} disabled={disabledReason !== ''}>
        {extra ? (
          <Box sx={{ display: 'flex', gap: space.gap, alignItems: 'center' }}>
            {control}
            {extra}
          </Box>
        ) : (
          control
        )}
      </DisabledReason>
    </SettingRow>
  )
}

export function PrefByKey({
  prefKey,
  extra,
  game,
  disabledReason,
}: {
  prefKey: string
  extra?: ReactNode
  game?: string
  disabledReason?: string
}) {
  const spec = specByKey(usePrefSpecs(), prefKey)
  if (!spec) {
    return null
  }
  return (
    <PrefRow
      spec={spec}
      extra={extra}
      {...(game ? { game } : {})}
      {...(disabledReason ? { disabledReason } : {})}
    />
  )
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
