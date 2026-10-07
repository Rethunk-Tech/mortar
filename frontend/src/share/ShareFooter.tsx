import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { MessageSquare } from 'lucide-react'
import { copyText } from './copyText.ts'
import type { ShownInfo } from './logic.ts'

export function ShareFooter({
  thunderstore,
  info,
  message,
}: {
  thunderstore: boolean
  info: ShownInfo
  message: string
}) {
  const { t } = useLingui()
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1.25,
        p: '14px 24px',
        borderTop: '1px solid var(--mortar-hairline-muted)',
      }}
    >
      <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
        {thunderstore ? t`Thunderstore games also offer an r2modman code.` : ''}
      </Typography>
      <Box sx={{ flex: 1 }} />
      <Button
        variant="outlined"
        color="inherit"
        startIcon={<MessageSquare size={16} />}
        disabled={info.count === 0 || info.tooLarge}
        onClick={() => copyText(message, t`Message copied`)}
        sx={{ height: 34 }}
      >
        {t`Copy as a message`}
      </Button>
    </Box>
  )
}
