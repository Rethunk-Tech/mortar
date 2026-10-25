import { Box, Button, Typography } from '@mui/material'
import { Hammer, Sparkles } from 'lucide-react'
import { useState } from 'react'
import { ChoicePicker } from '../../develop/ChoicePicker.tsx'
import { AppDialog, DialogFooter, ToneIcon } from './AppDialog.tsx'
import { ConfirmationDialog } from './ConfirmationDialog.tsx'
import { DerivedValue, FieldRow, FormDialog, FormSection, TextInput } from './FormDialog.tsx'

// For approving the dialog designs in the self-test sandbox (?gallery=dialogs); removed once every dialog uses them.
// Its sample text is English on purpose: it is not part of the app.

const MOD_NAME = /^[A-Z][A-Za-z0-9]{1,63}$/
const START_DESCRIPTION = 'The dropship lands somewhere new & different each delivery.'
const HEADING_PX = 22
const TEXT = {
  close: 'Close',
  heading: 'Dialog designs',
  notes:
    'A paragraph of release notes long enough to wrap, so the body scrolls while the header and footer stay put.',
}

const CATEGORIES = [
  { value: 'mods', label: 'Mods' },
  { value: 'bepinex', label: 'BepInEx' },
  { value: 'clientside', label: 'Client-side' },
  { value: 'serverside', label: 'Server-side' },
  { value: 'tweaks-and-quality-of-life', label: 'Tweaks & Quality Of Life' },
  { value: 'bug-fixes', label: 'Bug Fixes' },
  { value: 'performance', label: 'Performance' },
  { value: 'items', label: 'Items' },
  { value: 'equipment', label: 'Equipment' },
  { value: 'monsters', label: 'Monsters' },
  { value: 'moons', label: 'Moons' },
  { value: 'interiors', label: 'Interiors' },
  { value: 'hazards', label: 'Hazards' },
  { value: 'weather', label: 'Weather' },
  { value: 'vehicles', label: 'Vehicles' },
  { value: 'cosmetics', label: 'Cosmetics' },
  { value: 'suits', label: 'Suits' },
  { value: 'emotes', label: 'Emotes' },
  { value: 'furniture', label: 'Furniture' },
  { value: 'audio', label: 'Audio' },
  { value: 'translations', label: 'Translations' },
  { value: 'asset-replacements', label: 'Asset Replacements' },
  { value: 'libraries', label: 'Libraries' },
  { value: 'tools', label: 'Tools' },
  { value: 'misc', label: 'Misc' },
]

function EditExample({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [name, setName] = useState('ShopDrop')
  const [prefix, setPrefix] = useState('com.nomadtest')
  const [description, setDescription] = useState(START_DESCRIPTION)
  const [categories, setCategories] = useState([
    'mods',
    'bepinex',
    'clientside',
    'tweaks-and-quality-of-life',
    'items',
    'equipment',
    'monsters',
    'moons',
  ])
  const [author, setAuthor] = useState('NOMAD Test')
  const [team, setTeam] = useState('NomadTest')
  const nameError = MOD_NAME.test(name)
    ? ''
    : 'Start with a capital letter; letters and digits only.'
  const dirty = name !== 'ShopDrop' || author !== 'NOMAD Test' || description !== START_DESCRIPTION
  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title="Edit ShopDrop"
      subtitle="Saved into the project's own files."
      icon={
        <ToneIcon tone="neutral">
          <Hammer size={22} />
        </ToneIcon>
      }
      submitLabel="Save"
      issue={nameError ? 'Fix the mod name to save.' : ''}
      dirty={dirty}
      onSubmit={onClose}
    >
      <FormSection
        title="Identity"
        description="Renaming moves the project's folder, files and namespace."
      >
        <FieldRow>
          <TextInput label="Mod name" value={name} onChange={setName} error={nameError} />
          <TextInput label="Id prefix" value={prefix} onChange={setPrefix} />
        </FieldRow>
        <DerivedValue
          label="Plugin id"
          value={`${prefix}.${name}`}
          hint="BepInEx and other mods know it by this."
        />
      </FormSection>
      <FormSection title="Listing" description="What players see on Thunderstore.">
        <TextInput
          label="Description"
          value={description}
          onChange={setDescription}
          maxLength={250}
          multiline={true}
        />
        <ChoicePicker
          label="Categories"
          choices={CATEGORIES}
          value={categories}
          onChange={setCategories}
        />
      </FormSection>
      <FormSection title="Credits">
        <FieldRow>
          <TextInput
            label="Author"
            value={author}
            onChange={setAuthor}
            hint="Holds the copyright."
          />
          <TextInput label="Thunderstore team" value={team} onChange={setTeam} />
        </FieldRow>
      </FormSection>
    </FormDialog>
  )
}

function BaseExample({ open, onClose }: { open: boolean; onClose: () => void }) {
  return (
    <AppDialog
      open={open}
      onClose={onClose}
      title="What is new in 0.3.14"
      subtitle="Released October 25"
      icon={
        <ToneIcon tone="neutral">
          <Sparkles size={22} />
        </ToneIcon>
      }
      footer={
        <DialogFooter aside="Full notes on the website">
          <Button variant="contained" onClick={onClose}>
            {TEXT.close}
          </Button>
        </DialogFooter>
      }
    >
      {[
        'Develop tab',
        'Template options',
        'Known-broken lists',
        'Log suppressions',
        'Settings audit',
      ].map((h) => (
        <Box key={h} sx={{ mb: 2 }}>
          <Typography sx={{ fontWeight: 600, mb: 0.5 }}>{h}</Typography>
          <Typography sx={{ color: 'text.secondary', fontSize: 'medium.fontSize' }}>
            {TEXT.notes}
          </Typography>
        </Box>
      ))}
    </AppDialog>
  )
}

type Open = '' | 'form' | 'base' | 'confirm' | 'danger' | 'warning'

export function DialogGallery() {
  const [open, setOpen] = useState<Open>('')
  const [dontAsk, setDontAsk] = useState(false)
  const close = () => setOpen('')
  const show = (o: Open, label: string) => (
    <Button variant="outlined" onClick={() => setOpen(o)}>
      {label}
    </Button>
  )
  return (
    <Box
      sx={{
        p: 4,
        display: 'flex',
        flexDirection: 'column',
        gap: 2,
        bgcolor: 'background.default',
        minHeight: '100vh',
      }}
    >
      <Typography sx={{ fontSize: HEADING_PX, fontWeight: 600 }}>{TEXT.heading}</Typography>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
        {show('form', 'Form dialog')}
        {show('base', 'Base dialog')}
        {show('confirm', 'Confirmation')}
        {show('warning', 'Confirmation: warning')}
        {show('danger', 'Confirmation: danger')}
      </Box>
      <EditExample open={open === 'form'} onClose={close} />
      <BaseExample open={open === 'base'} onClose={close} />
      <ConfirmationDialog
        open={open === 'confirm'}
        title="Release ShopDrop 0.2.0?"
        body="Mortar adds the changelog entry, commits it and tags v0.2.0. Push them to publish."
        confirmLabel="Release"
        onConfirm={close}
        onCancel={close}
      />
      <ConfirmationDialog
        open={open === 'warning'}
        tone="warning"
        title="Update 3 mods while the game is running?"
        body="The game keeps the old versions until it restarts."
        confirmLabel="Update anyway"
        dontAskAgain={{ label: "Don't ask again", checked: dontAsk, onChange: setDontAsk }}
        onConfirm={close}
        onCancel={close}
      />
      <ConfirmationDialog
        open={open === 'danger'}
        tone="danger"
        title="Delete Lethal Company Development?"
        body="The profile goes to the trash; its mods stay in the store for other profiles."
        consequences={['12 mods and their settings', '3 save backups', 'Its change history']}
        confirmLabel="Delete profile"
        onConfirm={close}
        onCancel={close}
      />
    </Box>
  )
}
