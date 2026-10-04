import { useLingui } from '@lingui/react/macro'
import { Autocomplete, Box, TextField, Typography } from '@mui/material'
import { useCallback, useEffect, useRef, useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { MAX_ENTRY_NOTE, MAX_ENTRY_TAG, profileTags, takeTags } from './group.ts'
import { entryOf } from './lookup.ts'
import { heading } from './paper.ts'
import { useMods } from './store.ts'

const DEBOUNCE_MS = 400

export function ModNoteTags({ profile, mod }: { profile: Profile; mod: Mod }) {
  const { t } = useLingui()
  const setNoteTags = useMods((s) => s.setNoteTags)
  const entry = entryOf(profile, mod.key)
  const savedNote = entry?.note ?? ''
  const savedTags = entry?.tags ?? []
  const [note, setNote] = useState(savedNote)
  const [tags, setTags] = useState<string[]>(savedTags)
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined)
  const latest = useRef({ note: savedNote, tags: savedTags })
  const savedTagsKey = savedTags.join('\0')
  useEffect(() => {
    globalThis.clearTimeout(timer.current)
    setNote(savedNote)
    setTags(savedTagsKey === '' ? [] : savedTagsKey.split('\0'))
    latest.current = { note: savedNote, tags: savedTags }
  }, [savedNote, savedTags, savedTagsKey])
  const suggestions = profileTags(profile.entries)
  const save = useCallback(
    (nextNote: string, nextTags: string[]) => {
      globalThis.clearTimeout(timer.current)
      setNoteTags(mod, nextNote, nextTags).catch(reportUnexpected)
    },
    [mod, setNoteTags],
  )
  const schedule = (nextNote: string, nextTags: string[]) => {
    latest.current = { note: nextNote, tags: nextTags }
    globalThis.clearTimeout(timer.current)
    timer.current = globalThis.setTimeout(() => save(nextNote, nextTags), DEBOUNCE_MS)
  }
  useEffect(
    () => () => {
      globalThis.clearTimeout(timer.current)
      if (latest.current.note !== savedNote || latest.current.tags.join('\0') !== savedTagsKey) {
        save(latest.current.note, latest.current.tags)
      }
    },
    [savedNote, savedTagsKey, save],
  )
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.25 }}>
      <Box>
        <Typography sx={heading}>{t`Note`}</Typography>
        <TextField
          size="small"
          multiline={true}
          minRows={2}
          value={note}
          onChange={(e) => setNote(e.target.value)}
          onBlur={() => {
            if (note !== savedNote) {
              schedule(note, tags)
            }
          }}
          slotProps={{ htmlInput: { maxLength: MAX_ENTRY_NOTE, 'aria-label': t`Note` } }}
          sx={{ mt: 0.5, width: '100%' }}
        />
      </Box>
      <Box>
        <Typography sx={heading}>{t`Tags`}</Typography>
        <Autocomplete
          multiple={true}
          freeSolo={true}
          options={suggestions}
          value={tags}
          onChange={(_, next) => {
            const cleaned = takeTags(next)
            setTags(cleaned)
            schedule(note, cleaned)
          }}
          slotProps={{ chip: { size: 'small' } }}
          renderInput={(params) => (
            <TextField
              id={params.id}
              disabled={params.disabled}
              fullWidth={params.fullWidth}
              size="small"
              placeholder={tags.length === 0 ? t`Add a tag` : undefined}
              slotProps={{
                inputLabel: params.slotProps.inputLabel,
                input: params.slotProps.input,
                htmlInput: {
                  ...params.slotProps.htmlInput,
                  maxLength: MAX_ENTRY_TAG,
                  'aria-label': t`Tags`,
                },
              }}
            />
          )}
          sx={{ mt: 0.5 }}
        />
      </Box>
    </Box>
  )
}
