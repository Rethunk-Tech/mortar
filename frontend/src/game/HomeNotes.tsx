import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, TextField, Typography } from '@mui/material'
import { Check, NotebookPen } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { SetNotes } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { i18n } from '../i18n/index.ts'
import { useProfiles } from '../profiles/store.ts'
import { errorDetails } from '../toasts/errorKind.ts'
import { errorMessage } from '../toasts/report.ts'
import { HomePanel } from './HomePanel.tsx'

const DEBOUNCE_MS = 800
// Keep this in sync with profile.MaxNotes.
const MAX_NOTES = 20_000

const STATUS_TEXT: Record<'idle' | 'saving' | 'saved', () => string> = {
  idle: () => '',
  saving: () => i18n._(msg`Saving…`),
  saved: () => i18n._(msg`Saved`),
}

type Status = 'idle' | 'saving' | 'saved' | { error: string }

function NotesEditor({ profile, onDone }: { profile: Profile; onDone: () => void }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const replace = useProfiles((s) => s.replace)
  const [text, setText] = useState(profile.notes)
  const [status, setStatus] = useState<Status>('idle')
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
    setStatus('saving')
    SetNotes(game, profile.id, sent).then(
      (p) => {
        inflight.current = false
        replace(p)
        saved.current = sent
        setStatus('saved')
        save()
      },
      (error) => {
        inflight.current = false
        setStatus({ error: errorDetails(error) })
      },
    )
  }, [game, profile.id, replace])

  useEffect(() => {
    globalThis.addEventListener('blur', save)
    return () => {
      globalThis.removeEventListener('blur', save)
      save()
    }
  }, [save])

  return (
    <Box sx={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 1 }}>
      <TextField
        multiline={true}
        fullWidth={true}
        minRows={4}
        maxRows={14}
        autoFocus={true}
        value={text}
        onChange={(e) => {
          setText(e.target.value)
          latest.current = e.target.value
          globalThis.clearTimeout(timer.current)
          timer.current = globalThis.setTimeout(save, DEBOUNCE_MS)
        }}
        placeholder={t`Your notes for this profile, for example: Friday co-op profile. Keep everyone on the same version of the big mods before playing together. Notes travel in a shared .mortar file, not in a share link.`}
        slotProps={{
          htmlInput: { 'aria-label': t`Notes for ${profile.name}`, maxLength: MAX_NOTES },
        }}
      />
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
        <Button
          size="small"
          startIcon={<Check size={14} />}
          onClick={() => {
            save()
            onDone()
          }}
        >
          {t`Done`}
        </Button>
        {typeof status === 'object' ? (
          <Typography sx={{ fontSize: 13, color: 'error.main' }} title={errorDetails(status.error)}>
            {errorMessage(status.error)}
          </Typography>
        ) : (
          <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
            {STATUS_TEXT[status]?.()}
          </Typography>
        )}
      </Box>
    </Box>
  )
}

export function HomeNotes({ profile }: { profile: Profile }) {
  const { t } = useLingui()
  const [editing, setEditing] = useState(false)
  return (
    <HomePanel title={t`Notes`} span={2}>
      {editing ? (
        <NotesEditor profile={profile} onDone={() => setEditing(false)} />
      ) : (
        <>
          <Typography
            sx={{
              lineHeight: 1.5,
              whiteSpace: 'pre-wrap',
              overflowWrap: 'anywhere',
              color: profile.notes ? 'text.primary' : 'text.secondary',
            }}
          >
            {profile.notes || t`No notes yet.`}
          </Typography>
          <Button
            size="small"
            startIcon={<NotebookPen size={14} />}
            onClick={() => setEditing(true)}
          >
            {t`Edit notes`}
          </Button>
        </>
      )}
    </HomePanel>
  )
}
