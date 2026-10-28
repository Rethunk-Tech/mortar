import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import type { ReactNode } from 'react'
import type { PrefSpec } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { setOpenOverride } from '../profiles/openOverrides.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
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
import { saveSetting } from './persist.ts'
import { choiceStyle, prefControl } from './prefControl.ts'
import { prefCopy } from './prefCopy.ts'
import { specByKey, usePrefSpecs } from './prefSpecs.ts'
import { prefAsBool, prefAsNumber, prefAsString, prefRaw, specGameArg } from './prefValue.ts'
import { SettingRow } from './SettingsSection.tsx'
import { useSettings } from './store.ts'
import { useProfileScope } from './useProfileScope.ts'

type Options = { value: string; label: string }[]

// The control for a pref's kind and choices; save stores at once, commit is what a typed field awaits.
function prefControlNode(o: {
  spec: PrefSpec
  raw: unknown
  options: Options
  style: string
  label: string
  placeholder: string | undefined
  disabled: boolean
  save: (v: string) => void
  commit: (v: string) => Promise<void>
}): ReactNode {
  const { spec, raw, options, style, label, disabled, save, commit } = o
  const kind = prefControl(spec.type)
  let selectValue = prefAsString(raw, spec)
  if (options.length > 0 && !options.some((x) => x.value === selectValue)) {
    selectValue = options[0]?.value ?? selectValue
  }
  if (kind === 'switch') {
    return (
      <PrefSwitch
        checked={prefAsBool(raw, spec)}
        onChange={(on) => save(String(on))}
        label={label}
        disabled={disabled}
      />
    )
  }
  const choice = { value: selectValue, onChange: save, options, label, disabled }
  if (style === 'cards') {
    return <PrefCards {...choice} />
  }
  if (style === 'segmented' && options.length > 0) {
    return <PrefSegmented {...choice} />
  }
  if (kind === 'select' || options.length > 0) {
    return <PrefSelect {...choice} />
  }
  if (kind === 'number') {
    return (
      <PrefNumber
        value={prefAsNumber(raw, spec)}
        min={spec.min ?? 0}
        max={spec.max || Number.MAX_SAFE_INTEGER}
        onCommit={(n) => commit(String(n))}
        label={label}
        disabled={disabled}
      />
    )
  }
  return (
    <PrefText
      value={prefAsString(raw, spec)}
      {...(o.placeholder === undefined ? {} : { placeholder: o.placeholder })}
      onCommit={commit}
      label={label}
      disabled={disabled}
    />
  )
}

// On the open profile's scope a game setting reads and writes the profile's own value; one no profile can change
// shows the game's, held.
function useScopedPref(spec: PrefSpec, game: string | undefined) {
  const gameRaw = useSettings((s) => prefRaw(s, spec, game))
  const profileScope = useProfileScope() && spec.scope === 'game'
  const own = useProfiles((s) =>
    profileScope ? openProfileOf(s)?.overrides?.[spec.key] : undefined,
  )
  const ownValue = profileScope && spec.profileOverridable === true
  const gameArg = specGameArg(spec, game)
  return {
    gameRaw,
    own,
    ownValue,
    held: profileScope && !ownValue,
    raw: ownValue && own !== undefined ? own : gameRaw,
    commit: (value: string) =>
      ownValue && game
        ? setOpenOverride(game, spec.key, value)
        : SetByKey(spec.key, value, gameArg),
  }
}

// Beside a profile's own value: the game's, and a way back to it.
export function ScopeNote({
  own,
  gameLabel,
  onReset,
}: {
  own: string | undefined
  gameLabel: string
  onReset: () => void
}) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexShrink: 0 }}>
      <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }} noWrap={true}>
        {own === undefined ? t`Same as the game` : t`Game: ${{ value: gameLabel }}`}
      </Typography>
      {own === undefined ? null : (
        <Button size="small" onClick={onReset}>
          {t`Use the game's`}
        </Button>
      )}
    </Box>
  )
}

// Game-scope prefs render only where a game is passed, so global pages never write to a guessed game.
function PrefRow({
  spec,
  extra,
  game,
  disabledReason = '',
  hideTitle = false,
}: {
  spec: PrefSpec
  extra?: ReactNode
  game?: string
  disabledReason?: string
  hideTitle?: boolean
}) {
  const { t, i18n } = useLingui()
  const copy = prefCopy(i18n, spec.key)
  const scoped = useScopedPref(spec, game)
  if (spec.scope === 'game' && !game) {
    return null
  }
  const save = (value: string) => saveSetting(() => scoped.commit(value))
  const reason =
    disabledReason || (scoped.held ? t`Every profile of this game shares this one.` : '')
  const options = copy.options ?? (spec.values ?? []).map((value) => ({ value, label: value }))
  const style = options.length > 0 ? choiceStyle(options) : 'select'
  const control = prefControlNode({
    spec,
    raw: scoped.raw,
    options,
    style,
    label: copy.label,
    placeholder: copy.placeholder,
    disabled: reason !== '',
    save,
    commit: scoped.commit,
  })
  const gameValue = prefAsString(scoped.gameRaw, spec)
  const note = scoped.ownValue ? (
    <ScopeNote
      own={scoped.own}
      gameLabel={options.find((o) => o.value === gameValue)?.label ?? gameValue}
      onReset={() => game && saveSetting(() => setOpenOverride(game, spec.key, undefined))}
    />
  ) : null
  return (
    <SettingRow
      label={copy.label}
      description={copy.description}
      block={style === 'cards'}
      hideTitle={hideTitle}
    >
      <DisabledReason title={reason} disabled={reason !== ''}>
        {extra || note ? (
          <Box sx={{ display: 'flex', gap: space.gap, alignItems: 'center' }}>
            {control}
            {extra}
            {note}
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
  hideTitle,
}: {
  prefKey: string
  extra?: ReactNode
  game?: string
  disabledReason?: string
  // For a section whose heading already names the one row: the title stays for screen readers and search.
  hideTitle?: boolean
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
      {...(hideTitle ? { hideTitle } : {})}
    />
  )
}

/** Rows for keys; disabledBy names, per key, the other setting that holds it at its value and must change first. */
export function PrefKeys({
  keys,
  game,
  disabledBy = {},
}: {
  keys: string[]
  game?: string
  disabledBy?: Record<string, string>
}) {
  return (
    <>
      {keys.map((key) => (
        <PrefByKey
          key={key}
          prefKey={key}
          {...(game ? { game } : {})}
          {...(disabledBy[key] ? { disabledReason: disabledBy[key] } : {})}
        />
      ))}
    </>
  )
}
