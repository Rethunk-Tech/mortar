import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, TextField, Typography } from '@mui/material'
import { useCallback, useEffect, useRef, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { SetNotes } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { ago } from './ago.ts'

const DEBOUNCE_MS = 800
const TICK_MS = 60_000

type Status =
  | { kind: 'idle' }
  | { kind: 'saving' }
  | { kind: 'saved'; at: number }
  | { kind: 'error' }

function StatusLine({
  status,
  now,
  onRetry,
}: {
  status: Status
  now: number
  onRetry: () => void
}) {
  const { t } = useLingui()
  if (status.kind === 'idle') {
    return null
  }
  if (status.kind === 'error') {
    return (
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <Typography
          sx={{ fontSize: 13, color: 'error.main' }}
        >{t`Could not save notes`}</Typography>
        <Button size="small" onClick={onRetry} sx={{ whiteSpace: 'nowrap' }}>
          {t`Retry`}
        </Button>
      </Box>
    )
  }
  let label = t`Saving…`
  if (status.kind === 'saved') {
    const { unit, n } = ago(now - status.at)
    if (unit === 'now') {
      label = t`Saved · just now`
    } else if (unit === 'minute') {
      label = t`Saved · ${plural(n, { one: '# min ago', other: '# min ago' })}`
    } else if (unit === 'hour') {
      label = t`Saved · ${plural(n, { one: '# hour ago', other: '# hours ago' })}`
    } else {
      label = t`Saved · ${plural(n, { one: '# day ago', other: '# days ago' })}`
    }
  }
  return <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>{label}</Typography>
}

export function NotesTab({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const replace = useProfiles((s) => s.replace)
  const [text, setText] = useState(profile.notes)
  const [status, setStatus] = useState<Status>({ kind: 'idle' })
  const [now, setNow] = useState(() => Date.now())
  const latest = useRef(profile.notes)
  const saved = useRef(profile.notes)
  const inflight = useRef(false)
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined)

  const save = useCallback((): void => {
    globalThis.clearTimeout(timer.current)
    if (inflight.current || latest.current === saved.current) {
      return
    }
    const sent = latest.current
    inflight.current = true
    setStatus({ kind: 'saving' })
    SetNotes(game, profile.id, sent).then(
      (p) => {
        inflight.current = false
        replace(p)
        saved.current = sent
        setStatus({ kind: 'saved', at: Date.now() })
        save()
      },
      () => {
        inflight.current = false
        setStatus({ kind: 'error' })
      },
    )
  }, [game, profile.id, replace])

  useEffect(() => {
    const onBlur = () => save()
    globalThis.addEventListener('blur', onBlur)
    return () => {
      globalThis.removeEventListener('blur', onBlur)
      save()
    }
  }, [save])

  useEffect(() => {
    const id = globalThis.setInterval(() => setNow(Date.now()), TICK_MS)
    return () => globalThis.clearInterval(id)
  }, [])

  const onChange = (value: string) => {
    setText(value)
    latest.current = value
    globalThis.clearTimeout(timer.current)
    timer.current = globalThis.setTimeout(() => save(), DEBOUNCE_MS)
  }

  return (
    <Box sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1,
          minHeight: 30,
          px: 2,
          pt: 1.5,
          pb: 0.75,
        }}
      >
        <Box sx={{ flex: 1, minWidth: 0 }} />
        <StatusLine status={status} now={now} onRetry={() => save()} />
      </Box>
      <Box sx={{ flex: 1, minHeight: 0, display: 'flex', px: 2, pt: 0.75, pb: 2 }}>
        <TextField
          multiline={true}
          fullWidth={true}
          value={text}
          onChange={(e) => onChange(e.target.value)}
          placeholder={t`Your notes for this profile, for example: Friday co-op profile. Keep everyone on the same version of the big mods before playing together. Notes travel in a shared .mortar file, not in a share link.`}
          slotProps={{
            htmlInput: { 'aria-label': t`Notes for ${profile.name}` },
          }}
          sx={{
            flex: 1,
            minHeight: 0,
            '& .MuiInputBase-root': {
              flex: 1,
              alignItems: 'flex-start',
              overflow: 'auto',
              p: 2,
              fontSize: 15,
              lineHeight: 1.6,
              bgcolor: 'rgba(0,0,0,0.35)',
              borderRadius: '8px',
            },
          }}
        />
      </Box>
    </Box>
  )
}
