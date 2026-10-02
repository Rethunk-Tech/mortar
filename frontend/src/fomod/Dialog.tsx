import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Radio,
  RadioGroup,
  Typography,
} from '@mui/material'
import { useEffect, useMemo, useState } from 'react'
import { FomodImage } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { paper } from '../mods/paper.ts'
import { reportUnexpected } from '../toasts/report.ts'
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
      sx={{ maxWidth: 240, maxHeight: 160, objectFit: 'contain', display: 'block', mb: 1 }}
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

function groupOK(type: string, names: string[]): boolean {
  const n = names.length
  if (type === 'SelectExactlyOne') {
    return n === 1
  }
  if (type === 'SelectAtLeastOne') {
    return n >= 1
  }
  if (type === 'SelectAtMostOne') {
    return n <= 1
  }
  return true
}

function nextExclusive(groupType: string, plugin: string, on: boolean): string[] {
  if (on || groupType === 'SelectExactlyOne') {
    return [plugin]
  }
  return []
}

function FomodWizard({ session }: { session: FomodSession }) {
  const { t } = useLingui()
  const close = useFomod((s) => s.close)
  const install = useFomod((s) => s.install)
  const refresh = useFomod((s) => s.refresh)
  const [index, setIndex] = useState(0)
  const [choices, setChoices] = useState(() => savedChoices(session.ask))
  const [changed] = useState(Boolean(session.ask.changed))
  const steps = session.ask.steps ?? []
  const step = steps[index]
  const last = index >= steps.length - 1 || steps.length === 0
  const valid = useMemo(
    () =>
      (session.ask.steps ?? []).every((st) =>
        (st.groups ?? []).every((g) => groupOK(g.type, choices[st.name]?.[g.name] ?? [])),
      ),
    [session, choices],
  )

  const pick = (groupType: string, groupName: string, plugin: string, on: boolean) => {
    if (!step) {
      return
    }
    const current = choices[step.name]?.[groupName] ?? []
    let next: string[]
    if (groupType === 'SelectExactlyOne' || groupType === 'SelectAtMostOne') {
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
    <Dialog
      open={true}
      onClose={close}
      slotProps={{ paper }}
      transitionDuration={0}
      maxWidth="sm"
      fullWidth={true}
    >
      <DialogTitle>{session.ask.moduleName || t`Install options`}</DialogTitle>
      <DialogContent>
        {changed ? (
          <Typography variant="body2" color="warning.main" sx={{ mb: 2 }}>
            {t`This update changed its install options. Your earlier picks are kept where they still exist.`}
          </Typography>
        ) : null}
        {step ? (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Typography variant="subtitle2">{step.name}</Typography>
            {(step.groups ?? []).map((g) => {
              const selected = choices[step.name]?.[g.name] ?? []
              const exclusive = g.type === 'SelectExactlyOne' || g.type === 'SelectAtMostOne'
              const body = (g.plugins ?? []).map((p) => {
                const blocked = p.type === 'NotUsable'
                const checked = selected.includes(p.name) || p.type === 'Required'
                const control = exclusive ? (
                  <Radio
                    checked={checked}
                    disabled={blocked}
                    onChange={() => pick(g.type, g.name, p.name, true)}
                  />
                ) : (
                  <Checkbox
                    checked={checked}
                    disabled={blocked || p.type === 'Required'}
                    onChange={(_, on) => pick(g.type, g.name, p.name, on)}
                  />
                )
                return (
                  <Box key={p.name} sx={{ pl: 1 }}>
                    <FormControlLabel control={control} label={p.name} />
                    {p.image ? (
                      <PluginImage game={session.game} itemKey={session.key} rel={p.image} />
                    ) : null}
                    {p.description ? (
                      <Typography variant="body2" color="text.secondary" sx={{ pl: 4, mb: 1 }}>
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
                  {exclusive ? <RadioGroup>{body}</RadioGroup> : <Box>{body}</Box>}
                </Box>
              )
            })}
          </Box>
        ) : (
          <Typography>{t`This pack has files that always install.`}</Typography>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={close}>{t`Cancel`}</Button>
        <Button disabled={index === 0} onClick={() => setIndex((i) => i - 1)}>
          {t`Back`}
        </Button>
        {last ? (
          <Button
            variant="contained"
            disabled={!valid}
            onClick={() => install(choices).catch(reportUnexpected)}
          >
            {t`Install`}
          </Button>
        ) : (
          <Button variant="contained" onClick={() => setIndex((i) => i + 1)}>
            {t`Next`}
          </Button>
        )}
      </DialogActions>
    </Dialog>
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
