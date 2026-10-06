import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import type { ReactNode } from 'react'
import type { compareOpen } from './compareOpen.ts'

export function CompareSection({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Box sx={{ mb: 2 }}>
      <Typography sx={{ fontSize: 13, fontWeight: 700, mb: 0.75, color: 'text.secondary' }}>
        {title}
      </Typography>
      {children}
    </Box>
  )
}

export function CompareDiffRow({
  label,
  copyToB,
  copyToA,
  bName,
  aName,
  pending,
  open,
}: {
  label: string
  open?: ReturnType<typeof compareOpen>
  copyToB?: () => void
  copyToA?: () => void
  aName: string
  bName: string
  pending: boolean
}) {
  const { t } = useLingui()
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        py: 0.5,
        minHeight: 36,
      }}
    >
      <Typography
        sx={{
          flex: 1,
          fontSize: 14,
          minWidth: 0,
          ...(open ? { cursor: 'pointer', '&:hover': { textDecoration: 'underline' } } : {}),
        }}
        noWrap={true}
        title={label}
        {...open}
      >
        {label}
      </Typography>
      {copyToB ? (
        <Button
          size="small"
          variant="outlined"
          disabled={pending}
          onClick={copyToB}
          sx={{ flexShrink: 0 }}
        >
          {t`Copy to ${{ name: bName }}`}
        </Button>
      ) : null}
      {copyToA ? (
        <Button
          size="small"
          variant="outlined"
          disabled={pending}
          onClick={copyToA}
          sx={{ flexShrink: 0 }}
        >
          {t`Copy to ${{ name: aName }}`}
        </Button>
      ) : null}
    </Box>
  )
}
