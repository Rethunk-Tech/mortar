import { useLingui } from '@lingui/react/macro'
import { Box, Button, TextField, Typography } from '@mui/material'
import { useEffect, useRef, useState } from 'react'
import { useProfiles } from '../../profiles/store.ts'
import { ConfirmationDialog } from '../../shell/dialog/ConfirmationDialog.tsx'
import { space } from '../../theme/density.ts'
import { MONO } from '../../theme/theme.ts'
import { type InlineError, inlineError } from '../../toasts/report.ts'
import { LockedReason } from '../LockedReason.tsx'
import { useLocked } from '../useLocked.ts'
import { configApi } from './api.ts'
import type { Target } from './store.ts'

// A text file (YAML, TOML, XML, INI, JSON or plain text a mod keeps beside its .cfg) has no schema, so it is edited
// whole, and saved only when asked.
export function TextFile({
  target,
  name,
  onDirty,
}: {
  target: Target
  name: string
  onDirty: (dirty: boolean) => void
}) {
  const { t } = useLingui()
  const locked = useLocked()
  const [saved, setSaved] = useState<string | null>(null)
  const [text, setText] = useState('')
  const [error, setError] = useState<InlineError | null>(null)
  const { game, profile, key, id } = target
  useEffect(() => {
    const at = { game, profile, key, id }
    let live = true
    setSaved(null)
    setError(null)
    configApi
      .text(at, name)
      .then((v) => {
        if (live) {
          setSaved(v)
          setText(v)
        }
      })
      .catch((e: unknown) => live && setError(inlineError(e)))
    return () => {
      live = false
    }
  }, [game, profile, key, id, name])
  const adopt = (onDisk: string) => {
    setSaved(onDisk)
    setText(onDisk)
  }
  const [outside, setOutside] = useOutsideChange(target, name, { saved, text }, adopt)
  // A file edited elsewhere since it was read is not overwritten unseen: its new contents become the baseline.
  const save = () => {
    setError(null)
    configApi
      .text(target, name)
      .then(async (onDisk) => {
        if (onDisk !== saved) {
          setSaved(onDisk)
          setError({
            message: t`The file changed outside Mortar. Save again to replace it, or Revert to take its new contents.`,
            details: '',
          })
          return
        }
        await configApi.setText(target, name, text)
        setSaved(text)
        setOutside(null)
      })
      .catch((e: unknown) => setError(inlineError(e)))
  }
  const dirty = saved !== null && text !== saved
  useEffect(() => {
    onDirty(dirty)
    return () => onDirty(false)
  }, [dirty, onDirty])
  return (
    <Box
      sx={{
        flex: 1,
        minWidth: 0,
        minHeight: 0,
        display: 'flex',
        flexDirection: 'column',
        gap: space.gap,
        p: space.pad,
      }}
    >
      {error ? (
        <Typography
          role="alert"
          title={error.details}
          sx={{ fontSize: 'small.fontSize', color: 'error.main' }}
        >
          {saved === null
            ? t`Could not open: ${{ error: error.message }}`
            : t`Could not save: ${{ error: error.message }}`}
        </Typography>
      ) : null}
      {outside === null ? null : (
        <Typography role="status" sx={{ fontSize: 'small.fontSize', color: 'warning.main' }}>
          {t`The file changed outside Mortar.`}{' '}
          <Button
            size="small"
            onClick={() => {
              adopt(outside)
              setOutside(null)
            }}
          >
            {t`Reload`}
          </Button>
        </Typography>
      )}
      {/* MUI sizes a multiline textarea inline to its rows; it is forced to fill the pane instead and scrolls. */}
      <TextField
        multiline={true}
        rows={1}
        fullWidth={true}
        disabled={saved === null || locked}
        value={text}
        onChange={(e) => setText(e.target.value)}
        slotProps={{
          htmlInput: { 'aria-label': t`Contents of ${name}`, spellCheck: false },
        }}
        sx={{
          flex: 1,
          minHeight: 0,
          '& .MuiInputBase-root': { height: '100%', alignItems: 'stretch' },
          '& textarea': {
            height: '100% !important',
            overflow: 'auto',
            fontFamily: MONO,
            fontSize: 'base.fontSize',
          },
        }}
      />
      <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: space.gap }}>
        <Button
          variant="text"
          size="small"
          sx={{ mr: 'auto' }}
          disabled={saved === null}
          onClick={() =>
            configApi.openText(target, name).catch((e: unknown) => setError(inlineError(e)))
          }
        >
          {t`Open in editor`}
        </Button>
        <Button
          variant="outlined"
          size="small"
          disabled={!dirty}
          onClick={() => setText(saved ?? '')}
        >
          {t`Revert`}
        </Button>
        <LockedReason locked={locked}>
          <Button variant="contained" size="small" disabled={!dirty || locked} onClick={save}>
            {t`Save`}
          </Button>
        </LockedReason>
      </Box>
    </Box>
  )
}

// The game rewrites its files while Mortar is open: look again when the window regains focus or the profile changes.
// Unedited text takes the new contents; edited text keeps them, and the returned pair holds the contents on offer.
function useOutsideChange(
  target: Target,
  name: string,
  shown: { saved: string | null; text: string },
  adopt: (onDisk: string) => void,
) {
  const [outside, setOutside] = useState<string | null>(null)
  const { game, profile, key, id } = target
  const latest = useRef({ shown, adopt })
  latest.current = { shown, adopt }
  const updated = useProfiles((s) => s.profiles.find((p) => p.id === profile)?.updated)
  useEffect(() => {
    const at = { game, profile, key, id }
    let live = true
    setOutside(null)
    const look = () => {
      configApi
        .text(at, name)
        .then((onDisk) => {
          const { shown: now, adopt: take } = latest.current
          if (!live || now.saved === null || onDisk === now.saved) {
            return
          }
          if (now.text === now.saved) {
            take(onDisk)
            setOutside(null)
          } else {
            setOutside(onDisk)
          }
        })
        .catch(() => undefined)
    }
    if (updated !== undefined) {
      look()
    }
    window.addEventListener('focus', look)
    return () => {
      live = false
      window.removeEventListener('focus', look)
    }
  }, [game, profile, key, id, name, updated])
  return [outside, setOutside] as const
}

/** Asks before a file switch drops a text file's unsaved edits. */
export function DiscardTextDialog({
  open,
  name,
  onCancel,
  onConfirm,
}: {
  open: boolean
  name: string
  onCancel: () => void
  onConfirm: () => void
}) {
  const { t } = useLingui()
  return (
    <ConfirmationDialog
      open={open}
      title={t`Discard your changes?`}
      body={t`${name} has edits that are not saved.`}
      confirmLabel={t`Discard`}
      tone="warning"
      onCancel={onCancel}
      onConfirm={onConfirm}
    />
  )
}
