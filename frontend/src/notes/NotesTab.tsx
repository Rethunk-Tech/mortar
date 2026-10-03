import { useLingui } from '@lingui/react/macro'
import { Box, Button, TextField, Typography } from '@mui/material'
import { useCallback, useEffect, useRef, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { SetNotes } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { useNow } from '../i18n/useNow.ts'
import { useProfiles } from '../profiles/store.ts'
import { errorDetails, errorMessage } from '../toasts/report.ts'

const DEBOUNCE_MS = 800
// Keep this in sync with profile.MaxNotes.
const MAX_NOTES = 20_000
const COUNTER_THRESHOLD = 500

type Status =
  | { kind: 'idle' }
  | { kind: 'saving' }
  | { kind: 'saved'; at: number }
  | { kind: 'error'; message: string }

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
        <Typography sx={{ fontSize: 13, color: 'error.main' }} title={errorDetails(status.message)}>
          {errorMessage(status.message)}
        </Typography>
        <Button size="small" onClick={onRetry} sx={{ whiteSpace: 'nowrap' }}>
          {t`Retry`}
        </Button>
      </Box>
    )
  }
  let label = t`Saving…`
  if (status.kind === 'saved') {
    label = t`Saved · ${formatWhen(status.at, { now })}`
  }
  return <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>{label}</Typography>
}

export function NotesTab({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const replace = useProfiles((s) => s.replace)
  const [text, setText] = useState(profile.notes)
  const [status, setStatus] = useState<Status>({ kind: 'idle' })
  const now = useNow()
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
      (error) => {
        inflight.current = false
        setStatus({
          kind: 'error',
          message: error instanceof Error ? error.message : String(error),
        })
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
          position: 'relative',
          flex: 1,
          minHeight: 0,
          display: 'flex',
          px: 2,
          pt: 1.5,
          pb: 1.5,
        }}
      >
        <TextField
          multiline={true}
          fullWidth={true}
          value={text}
          onChange={(e) => onChange(e.target.value)}
          placeholder={t`Your notes for this profile, for example: Friday co-op profile. Keep everyone on the same version of the big mods before playing together. Notes travel in a shared .mortar file, not in a share link.`}
          slotProps={{
            htmlInput: { 'aria-label': t`Notes for ${profile.name}`, maxLength: MAX_NOTES },
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
        {text.length >= MAX_NOTES - COUNTER_THRESHOLD ? (
          <Typography
            sx={{ position: 'absolute', right: 32, top: 8, fontSize: 12, color: 'text.secondary' }}
          >
            {`${text.length}/${MAX_NOTES}`}
          </Typography>
        ) : null}
        {/* Floats in the field's corner so the save status takes no room from the notes. */}
        <Box sx={{ position: 'absolute', right: 32, bottom: 20 }}>
          <StatusLine status={status} now={now} onRetry={() => save()} />
        </Box>
      </Box>
    </Box>
  )
}
