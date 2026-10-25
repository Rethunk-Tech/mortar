import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Autocomplete,
  FormControlLabel,
  Radio,
  RadioGroup,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material'
import { Tag, Tags } from 'lucide-react'
import { useId } from 'react'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { Field, FormDialog } from '../shell/dialog/FormDialog.tsx'
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
  const tagId = useId()
  const apply = () => {
    if (tag.trim() === '') {
      return
    }
    onClose()
    setTagMany(mods, tag, addTag).catch(reportUnexpected)
  }
  return (
    <FormDialog
      open={open}
      onClose={onClose}
      size="sm"
      title={plural({ count: mods.length }, { one: 'Tag # mod', other: 'Tag # mods' })}
      icon={
        <ToneIcon tone="neutral">
          <Tag size={22} />
        </ToneIcon>
      }
      submitLabel={t`Apply`}
      issue={tag.trim() === '' ? t`Type a tag.` : ''}
      dirty={tag !== ''}
      onSubmit={apply}
    >
      <Field label={t`Tag`} htmlFor={tagId}>
        <Autocomplete
          id={tagId}
          freeSolo={true}
          options={tags}
          value={tag}
          onInputChange={(_, value) => setTag(value)}
          renderInput={(params) => <TextField {...params} size="small" autoFocus={true} />}
        />
      </Field>
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
        sx={{ alignSelf: 'flex-start' }}
      >
        <ToggleButton value="add">{t`Add`}</ToggleButton>
        <ToggleButton value="remove">{t`Remove`}</ToggleButton>
      </ToggleButtonGroup>
    </FormDialog>
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
    <FormDialog
      open={open}
      onClose={onClose}
      size="sm"
      title={plural(
        { count: mods.length },
        { one: 'Set category for # mod', other: 'Set category for # mods' },
      )}
      icon={
        <ToneIcon tone="neutral">
          <Tags size={22} />
        </ToneIcon>
      }
      submitLabel={t`Apply`}
      dirty={false}
      onSubmit={() => {
        onClose()
        setCategoryMany(mods, category).catch(reportUnexpected)
      }}
    >
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
    </FormDialog>
  )
}
