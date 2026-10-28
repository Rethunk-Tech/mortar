import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, FormControlLabel, MenuItem, MenuList } from '@mui/material'
import { FolderPlus } from 'lucide-react'
import { useState } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { AppDialog, DialogFooter, ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { TextInput } from '../shell/dialog/FormDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { requirementKeys } from './dependents.ts'
import { LockedReason } from './LockedReason.tsx'
import { useMods } from './store.ts'
import { addModsToGroup } from './storeGroups.ts'
import { useLocked } from './useLocked.ts'

const ICON_SIZE = 16
const MAX_GROUP_NAME = 60

function AddToGroupMenuItem({ locked, onClick }: { locked: boolean; onClick: () => void }) {
  const { t } = useLingui()
  return (
    <LockedReason locked={locked}>
      <MenuAction
        disabled={locked}
        icon={<FolderPlus size={ICON_SIZE} />}
        label={t`Add to group…`}
        onClick={onClick}
      />
    </LockedReason>
  )
}

// Adds the mods, and by default what they need, to a group the profile has or a new one.
function AddToGroupDialog({
  open,
  onClose,
  mods,
  profile,
}: {
  open: boolean
  onClose: () => void
  mods: Mod[]
  profile: Profile | undefined
}) {
  const { t } = useLingui()
  const locked = useLocked()
  const [name, setName] = useState('')
  const [withNeeds, setWithNeeds] = useState(true)
  const all = useMods((s) => s.mods)
  const needed = open ? requirementKeys(all, mods) : []
  const groups = profile?.groups ?? []
  const add = async (groupName: string) => {
    const trimmed = groupName.trim()
    if (trimmed === '' || locked) {
      return
    }
    const keys = [...new Set(mods.map((m) => m.key))]
    await addModsToGroup(withNeeds ? [...keys, ...needed] : keys, trimmed)
    setName('')
    onClose()
  }
  return (
    <AppDialog
      open={open}
      onClose={onClose}
      size="sm"
      title={t`Add to group…`}
      icon={
        <ToneIcon tone="neutral">
          <FolderPlus size={22} />
        </ToneIcon>
      }
      footer={
        <DialogFooter>
          <Button onClick={onClose}>{t`Cancel`}</Button>
          <LockedReason locked={locked}>
            <Button
              variant="contained"
              disabled={locked || name.trim() === ''}
              onClick={() => add(name).catch(reportUnexpected)}
            >
              {t`Add`}
            </Button>
          </LockedReason>
        </DialogFooter>
      }
    >
      <LockedReason locked={locked}>
        {groups.length === 0 ? (
          <EmptyState compact={true} icon={<FolderPlus size={28} />} title={t`No groups yet`}>
            {t`Name a group below to create it.`}
          </EmptyState>
        ) : (
          <MenuList dense={true}>
            {groups.map((g) => (
              <MenuItem
                key={g.name}
                disabled={locked}
                onClick={() => {
                  add(g.name ?? '').catch(reportUnexpected)
                }}
              >
                {g.name}
              </MenuItem>
            ))}
          </MenuList>
        )}
        {/* Enter in the field submits; the keydown bubbles up from the input. */}
        <Box
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              add(name).catch(reportUnexpected)
            }
          }}
        >
          <TextInput
            label={t`New group`}
            value={name}
            onChange={setName}
            maxLength={MAX_GROUP_NAME}
            autoFocus={true}
            disabled={locked}
          />
        </Box>
        {needed.length > 0 ? (
          <FormControlLabel
            control={<Checkbox checked={withNeeds} onChange={(_, on) => setWithNeeds(on)} />}
            label={plural(
              { count: needed.length },
              {
                one: 'Also add the # mod they need',
                other: 'Also add the # mods they need',
              },
            )}
          />
        ) : null}
      </LockedReason>
    </AppDialog>
  )
}

export { AddToGroupDialog, AddToGroupMenuItem }
