import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Checkbox,
  FormControlLabel,
  Radio,
  RadioGroup,
  Typography,
} from '@mui/material'
import { useEffect, useMemo, useState } from 'react'
import type { FomodStep } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { FomodImage } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { AppDialog, DialogFooter } from '../shell/dialog/AppDialog.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { type FomodSession, useFomod, watchFomodQueue } from './store.ts'

function bytesOf(data: unknown): Uint8Array {
  if (data instanceof Uint8Array) {
    return data
  }
  if (data instanceof ArrayBuffer) {
    return new Uint8Array(data)
  }
  if (Array.isArray(data)) {
    return Uint8Array.from(data)
  }
  return new Uint8Array()
}

function PluginImage({ game, itemKey, rel }: { game: string; itemKey: string; rel: string }) {
  const [url, setUrl] = useState('')
  useEffect(() => {
    let revoke = ''
    let alive = true
    setUrl('')
    FomodImage(game, itemKey, rel)
      .then((data) => {
        if (!alive) {
          return
        }
        const raw = bytesOf(data)
        const copy = new ArrayBuffer(raw.byteLength)
        new Uint8Array(copy).set(raw)
        revoke = URL.createObjectURL(new Blob([copy]))
        setUrl(revoke)
      })
      // Authors often name preview images that are not in the archive; the step is shown without one.
      .catch(() => undefined)
    return () => {
      alive = false
      if (revoke) {
        URL.revokeObjectURL(revoke)
      }
    }
  }, [game, itemKey, rel])
  if (!url) {
    return null
  }
  return (
    <Box
      component="img"
      src={url}
      alt=""
      sx={{ maxWidth: 240, maxHeight: 160, objectFit: 'contain', display: 'block', mb: space.gap }}
    />
  )
}

function savedChoices(ask: FomodSession['ask']): Record<string, Record<string, string[]>> {
  const out: Record<string, Record<string, string[]>> = {}
  for (const [step, groups] of Object.entries(ask.choices ?? {})) {
    for (const [group, names] of Object.entries(groups ?? {})) {
      out[step] = { ...out[step], [group]: names ?? [] }
    }
  }
  for (const step of ask.steps ?? []) {
    for (const group of step.groups ?? []) {
      const current = out[step.name]?.[group.name] ?? []
      const required = (group.plugins ?? [])
        .filter((p) => is(p.type, 'Required'))
        .map((p) => p.name)
      const recommended =
        current.length === 0
          ? (group.plugins ?? []).filter((p) => is(p.type, 'Recommended')).map((p) => p.name)
          : []
      const seeded = [...new Set([...current, ...required, ...recommended])]
      if (seeded.length > 0) {
        out[step.name] = { ...out[step.name], [group.name]: seeded }
      }
    }
  }
  return out
}

function setGroup(
  choices: Record<string, Record<string, string[]>>,
  step: string,
  group: string,
  names: string[],
): Record<string, Record<string, string[]>> {
  return { ...choices, [step]: { ...choices[step], [group]: names } }
}

// Packs write these in any case; the installer compares them case-insensitively.
const is = (type: string, expected: string): boolean =>
  type.toLowerCase() === expected.toLowerCase()

function groupOK(type: string, names: string[]): boolean {
  const n = names.length
  if (is(type, 'SelectExactlyOne')) {
    return n === 1
  }
  if (is(type, 'SelectAtLeastOne')) {
    return n >= 1
  }
  if (is(type, 'SelectAtMostOne')) {
    return n <= 1
  }
  return true
}

type Choices = Record<string, Record<string, string[]>>

function invalidGroupOf(step: FomodStep, choices: Choices): string | undefined {
  return (step.groups ?? []).find((g) => !groupOK(g.type, choices[step.name]?.[g.name] ?? []))?.name
}

function nextExclusive(groupType: string, plugin: string, on: boolean): string[] {
  if (on || is(groupType, 'SelectExactlyOne')) {
    return [plugin]
  }
  return []
}

const descriptionSx = { pl: 4, mb: space.gap }

function FomodWizard({ session }: { session: FomodSession }) {
  const { t } = useLingui()
  const close = useFomod((s) => s.close)
  const install = useFomod((s) => s.install)
  const refresh = useFomod((s) => s.refresh)
  const [pending, run] = usePending()
  const [index, setIndex] = useState(0)
  const [choices, setChoices] = useState(() => savedChoices(session.ask))
  const [changed] = useState(Boolean(session.ask.changed))
  const steps = session.ask.steps ?? []
  useEffect(() => {
    setIndex((current) => Math.min(current, Math.max(0, steps.length - 1)))
  }, [steps.length])
  const step = steps[index]
  const last = index >= steps.length - 1 || steps.length === 0
  const valid = useMemo(
    () => steps.every((st) => invalidGroupOf(st, choices) === undefined),
    [steps, choices],
  )
  const invalidGroup = step && invalidGroupOf(step, choices)

  const pick = (groupType: string, groupName: string, plugin: string, on: boolean) => {
    if (!step) {
      return
    }
    const current = choices[step.name]?.[groupName] ?? []
    let next: string[]
    if (is(groupType, 'SelectExactlyOne') || is(groupType, 'SelectAtMostOne')) {
      next = nextExclusive(groupType, plugin, on)
    } else if (on) {
      next = [...current.filter((n) => n !== plugin), plugin]
    } else {
      next = current.filter((n) => n !== plugin)
    }
    const updated = setGroup(choices, step.name, groupName, next)
    setChoices(updated)
    refresh(updated).catch(reportUnexpected)
  }

  return (
    <AppDialog
      open={true}
      onClose={close}
      closable={false}
      size="md"
      title={session.ask.moduleName || t`Install options`}
      footer={
        <DialogFooter>
          <Button onClick={close}>{t`Cancel`}</Button>
          <Button disabled={index === 0} onClick={() => setIndex((i) => i - 1)}>
            {t`Back`}
          </Button>
          {last ? (
            <Button
              variant="contained"
              disabled={!valid || pending}
              onClick={() => run(() => install(choices))}
            >
              {t`Install`}
            </Button>
          ) : (
            <Button
              variant="contained"
              disabled={!step || Boolean(invalidGroup)}
              onClick={() => setIndex((i) => i + 1)}
            >
              {t`Next`}
            </Button>
          )}
        </DialogFooter>
      }
    >
      {changed ? (
        <Typography variant="body2" color="warning.main" sx={{ mb: 2 }}>
          {t`This update changed its install options. Your earlier picks are kept where they still exist.`}
        </Typography>
      ) : null}
      {step ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.pad }}>
          <Typography variant="subtitle2">
            {t`Step ${{ step: index + 1 }} of ${{ count: steps.length }}: ${{ name: step.name }}`}
          </Typography>
          {(step.groups ?? []).map((g) => {
            const selected = choices[step.name]?.[g.name] ?? []
            const exclusive = is(g.type, 'SelectExactlyOne') || is(g.type, 'SelectAtMostOne')
            const all = is(g.type, 'SelectAll')
            const body = (g.plugins ?? []).map((p) => {
              const blocked = is(p.type, 'NotUsable')
              const checked = selected.includes(p.name) || is(p.type, 'Required') || all
              const control = (
                <DisabledReason title={t`This option cannot be used.`} disabled={blocked}>
                  {exclusive ? (
                    <Radio
                      checked={checked}
                      disabled={blocked}
                      onChange={() => pick(g.type, g.name, p.name, true)}
                      onClick={() => {
                        if (checked && is(g.type, 'SelectAtMostOne')) {
                          pick(g.type, g.name, p.name, false)
                        }
                      }}
                    />
                  ) : (
                    <Checkbox
                      checked={checked}
                      disabled={blocked || all || is(p.type, 'Required')}
                      onChange={(_, on) => pick(g.type, g.name, p.name, on)}
                    />
                  )}
                </DisabledReason>
              )
              return (
                <Box key={p.name} sx={{ pl: space.gap }}>
                  <FormControlLabel control={control} label={p.name} />
                  {p.image ? (
                    <PluginImage game={session.game} itemKey={session.key} rel={p.image} />
                  ) : null}
                  {p.description ? (
                    <Typography variant="body2" color="text.secondary" sx={descriptionSx}>
                      {p.description}
                    </Typography>
                  ) : null}
                </Box>
              )
            })
            return (
              <Box key={g.name}>
                <Typography variant="body2" color="text.secondary">
                  {g.name}
                </Typography>
                {exclusive ? (
                  <RadioGroup aria-label={g.name}>{body}</RadioGroup>
                ) : (
                  <Box role="group" aria-label={g.name}>
                    {body}
                  </Box>
                )}
              </Box>
            )
          })}
        </Box>
      ) : (
        <Typography>{t`This pack has files that always install.`}</Typography>
      )}
      {invalidGroup ? (
        <Typography color="error" variant="body2" sx={{ mt: 2 }}>
          {t`Choose an option in ${invalidGroup}.`}
        </Typography>
      ) : null}
    </AppDialog>
  )
}

export function FomodDialog() {
  const session = useFomod((s) => s.session)
  useEffect(() => {
    watchFomodQueue()
  }, [])
  if (!session) {
    return null
  }
  return <FomodWizard key={`${session.key}:${session.queueId ?? ''}`} session={session} />
}
