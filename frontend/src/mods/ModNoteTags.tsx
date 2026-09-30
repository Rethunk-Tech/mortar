import { useLingui } from '@lingui/react/macro'
import { Autocomplete, Box, TextField, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { MAX_ENTRY_NOTE, MAX_ENTRY_TAG, profileTags, takeTags } from './group.ts'
import { entryOf } from './lookup.ts'
import { heading } from './paper.ts'
import { useMods } from './store.ts'

export function ModNoteTags({ profile, mod }: { profile: Profile; mod: Mod }) {
  const { t } = useLingui()
  const setNoteTags = useMods((s) => s.setNoteTags)
  const entry = entryOf(profile, mod.key)
  const savedNote = entry?.note ?? ''
  const savedTags = entry?.tags ?? []
  const [note, setNote] = useState(savedNote)
  const [tags, setTags] = useState<string[]>(savedTags)
  const savedTagsKey = savedTags.join('\0')
  useEffect(() => {
    setNote(savedNote)
    setTags(savedTagsKey === '' ? [] : savedTagsKey.split('\0'))
  }, [savedNote, savedTagsKey])
  const suggestions = profileTags(profile.entries)
  const save = (nextNote: string, nextTags: string[]) => {
    setNoteTags(mod, nextNote, nextTags).catch(reportUnexpected)
  }
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
              save(note, tags)
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
            save(note, cleaned)
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
