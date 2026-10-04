import { useLingui } from '@lingui/react/macro'
import {
  Autocomplete,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { reportUnexpected } from '../toasts/report.ts'

export function TagSelectionDialog({
  open,
  onClose,
  mods,
  tags,
  setTag,
  tag,
  addTag,
  setAddTag,
  setTagMany,
}: {
  open: boolean
  onClose: () => void
  mods: Mod[]
  tags: string[]
  setTag: (value: string) => void
  tag: string
  addTag: boolean
  setAddTag: (value: boolean) => void
  setTagMany: (mods: Mod[], tag: string, add: boolean) => Promise<void>
}) {
  const { t } = useLingui()
  return (
    <Dialog open={open} onClose={onClose}>
      <DialogTitle>{t`Tag selected mods`}</DialogTitle>
      <DialogContent sx={{ minWidth: 320, pt: 2 }}>
        <Autocomplete
          freeSolo={true}
          options={tags}
          value={tag}
          onInputChange={(_, value) => setTag(value)}
          renderInput={(params) => <TextField {...params} autoFocus={true} label={t`Tag`} />}
        />
        <ToggleButtonGroup
          aria-label={t`Add or remove the tag`}
          exclusive={true}
          size="small"
          value={addTag ? 'add' : 'remove'}
          onChange={(_, value: string | null) => {
            if (value === 'add') {
              setAddTag(true)
            }
            if (value === 'remove') {
              setAddTag(false)
            }
          }}
          sx={{ mt: 1 }}
        >
          <ToggleButton value="add">{t`Add`}</ToggleButton>
          <ToggleButton value="remove">{t`Remove`}</ToggleButton>
        </ToggleButtonGroup>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Cancel`}</Button>
        <Button
          variant="contained"
          disabled={tag.trim() === ''}
          onClick={() => {
            onClose()
            setTagMany(mods, tag, addTag).catch(reportUnexpected)
          }}
        >
          {t`Apply`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export function CategorySelectionDialog({
  open,
  onClose,
  mods,
  categories,
  category,
  setCategory,
  setCategoryMany,
}: {
  open: boolean
  onClose: () => void
  mods: Mod[]
  categories: { id: string; name: string }[]
  category: string
  setCategory: (value: string) => void
  setCategoryMany: (mods: Mod[], category: string) => Promise<void>
}) {
  const { t } = useLingui()
  return (
    <Dialog open={open} onClose={onClose}>
      <DialogTitle>{t`Set category for selected mods`}</DialogTitle>
      <DialogContent sx={{ minWidth: 280, pt: 2 }}>
        {[{ id: '', name: t`Uncategorized` }, ...categories].map((option) => (
          <Button
            key={option.id}
            fullWidth={true}
            variant={category === option.id ? 'contained' : 'text'}
            onClick={() => setCategory(option.id)}
            sx={{ justifyContent: 'flex-start' }}
          >
            {option.name}
          </Button>
        ))}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Cancel`}</Button>
        <Button
          variant="contained"
          onClick={() => {
            onClose()
            setCategoryMany(mods, category).catch(reportUnexpected)
          }}
        >
          {t`Apply`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
