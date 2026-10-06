import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Badge, Box, ButtonBase, Tooltip } from '@mui/material'
import { Download } from 'lucide-react'
import { useQueue } from './store.ts'
import { totals } from './totals.ts'

// The queue's entry in the sidebar's bottom block, beside the Support button.
export function QueueButton() {
  const { t } = useLingui()
  const left = useQueue((s) => totals(s.state.items).left)
  const setOpen = useQueue((s) => s.setOpen)
  return (
    <Box sx={{ display: 'flex', flex: 1, pl: '6px' }}>
      <Tooltip title={t`Downloads`}>
        <ButtonBase
          onClick={() => setOpen(true)}
          aria-label={
            left > 0
              ? plural(left, { one: 'Downloads, # left', other: 'Downloads, # left' })
              : t`Downloads`
          }
          sx={{
            width: 40,
            height: 40,
            borderRadius: '6px',
            '&:hover': { bgcolor: 'action.hover' },
          }}
        >
          <Badge
            // None rather than 0: a hidden badge still holds its 0, text the button's name would then lack.
            badgeContent={left || null}
            color="primary"
            max={99}
            slotProps={{ badge: { 'aria-hidden': true } }}
          >
            <Download size={18} aria-hidden={true} />
          </Badge>
        </ButtonBase>
      </Tooltip>
    </Box>
  )
}
