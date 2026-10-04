import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Autocomplete,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Radio,
  RadioGroup,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
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
  const apply = () => {
    if (tag.trim() === '') {
      return
    }
    onClose()
    setTagMany(mods, tag, addTag).catch(reportUnexpected)
  }
  return (
    <Dialog open={open} onClose={onClose}>
      <form
        onSubmit={(e) => {
          e.preventDefault()
          apply()
        }}
      >
        <DialogTitle>{plural(mods.length, { one: 'Tag # mod', other: 'Tag # mods' })}</DialogTitle>
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
          <Button type="submit" variant="contained" disabled={tag.trim() === ''}>
            {t`Apply`}
          </Button>
        </DialogActions>
      </form>
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
      <DialogTitle>
        {plural(mods.length, { one: 'Set category for # mod', other: 'Set category for # mods' })}
      </DialogTitle>
      <DialogContent sx={{ minWidth: 280, pt: 2 }}>
        <RadioGroup
          aria-label={t`Category`}
          value={category}
          onChange={(e) => setCategory(e.target.value)}
        >
          {[{ id: '', name: t`Uncategorised` }, ...categories].map((option) => (
            <FormControlLabel
              key={option.id}
              value={option.id}
              control={<Radio />}
              label={option.name}
            />
          ))}
        </RadioGroup>
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
