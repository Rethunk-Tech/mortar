import { useLingui } from '@lingui/react/macro'
import { Badge, Box, ButtonBase } from '@mui/material'
import { Download } from 'lucide-react'
import { compact } from '../game/compact.ts'
import { useQueue } from './store.ts'
import { totals } from './totals.ts'

// The queue's entry in the sidebar's bottom block: an icon row that grows a Support button beside it later.
export function QueueButton() {
  const { t } = useLingui()
  const left = useQueue((s) => totals(s.state.items).left)
  const setOpen = useQueue((s) => s.setOpen)
  return (
    <Box sx={{ display: 'flex', p: '6px', [compact]: { p: 0, justifyContent: 'center' } }}>
      <ButtonBase
        onClick={() => setOpen(true)}
        aria-label={t`Downloads`}
        sx={{
          flex: 1,
          height: 36,
          justifyContent: 'flex-start',
          gap: 1.25,
          px: '10px',
          borderRadius: '6px',
          fontFamily: 'inherit',
          fontSize: 14,
          whiteSpace: 'nowrap',
          '&:hover': { bgcolor: 'action.hover' },
          [compact]: { flex: 'none', width: 40, height: 40, justifyContent: 'center', p: 0 },
        }}
      >
        <Badge badgeContent={left} color="primary" max={99}>
          <Download size={18} aria-hidden={true} />
        </Badge>
        <Box component="span" sx={{ [compact]: { display: 'none' } }}>
          {t`Downloads`}
        </Box>
      </ButtonBase>
    </Box>
  )
}
