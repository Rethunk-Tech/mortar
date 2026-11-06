import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Button } from '@mui/material'
import type { ReactNode } from 'react'
import { formatWhen } from '../i18n/formatWhen.ts'
import { openSettings } from '../nav/store.ts'
import { SAVED_OPACITY } from './savedOpacity.ts'

/** Marks data a site gave before its sign-in ended: shown from the saved copy, refreshed once the player signs in. */
export function SavedCopyNote({
  at,
}: {
  at?: Parameters<typeof formatWhen>[0] | null | undefined
}) {
  const { t } = useLingui()
  return (
    <Alert
      variant="compact"
      severity="info"
      action={
        <Button color="inherit" size="small" onClick={() => openSettings('accounts')}>
          {t`Sign in…`}
        </Button>
      }
    >
      {at
        ? t`Saved ${{ when: formatWhen(at) }}. Sign in to refresh it.`
        : t`Saved from an earlier visit. Sign in to refresh it.`}
    </Alert>
  )
}

/** Data a site gave before its sign-in ended, dimmed under its note; as it is while `saved` is false. */
export function SavedCopy({
  saved = true,
  at,
  children,
}: {
  saved?: boolean
  at?: Parameters<typeof formatWhen>[0] | null | undefined
  children: ReactNode
}) {
  if (!saved) {
    return children
  }
  return (
    <>
      <SavedCopyNote at={at} />
      <Box
        sx={{ display: 'flex', flexDirection: 'column', gap: 'inherit', opacity: SAVED_OPACITY }}
      >
        {children}
      </Box>
    </>
  )
}
