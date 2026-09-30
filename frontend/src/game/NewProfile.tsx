import { useLingui } from '@lingui/react/macro'
import { Popover } from '@mui/material'
import { useProfiles } from '../profiles/store.ts'
import { NameField } from './NameField.tsx'

export function NewProfilePopover({
  anchor,
  onClose,
}: {
  anchor: HTMLElement | null
  onClose: () => void
}) {
  const { t } = useLingui()
  const create = useProfiles((s) => s.create)
  return (
    <Popover
      open={anchor !== null}
      anchorEl={anchor}
      onClose={onClose}
      anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
      slotProps={{ paper: { sx: { p: 1 } } }}
    >
      <NameField
        initial=""
        label={t`Profile name`}
        size="small"
        onSubmit={create}
        onCancel={onClose}
      />
    </Popover>
  )
}
