import { Button, type ButtonProps } from '@mui/material'
import type { ReactNode } from 'react'
import { openPageReported } from './openPageReported.ts'

export function PageLinkButton({
  url,
  sx,
  children,
}: {
  url: string
  sx?: ButtonProps['sx']
  children: ReactNode
}) {
  return (
    <Button
      size="small"
      {...(sx === undefined ? {} : { sx })}
      onClick={() => openPageReported(url)}
    >
      {children}
    </Button>
  )
}
