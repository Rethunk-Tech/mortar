import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { LogIn } from 'lucide-react'
import { SourceLogo } from '../brand/sources/SourceLogo.tsx'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { openSettings } from '../nav/store.ts'
import { EmptyState } from './EmptyState.tsx'

const LOGO_PX = 40

/** In the space the content would fill: what a site shares only with a signed-in account, and the way to sign in. */
export function SignInPrompt({
  source,
  children,
}: {
  source?: string | undefined
  children?: string | undefined
}) {
  const { t } = useLingui()
  const site = source ? sourceLabel(source) : ''
  return (
    <EmptyState
      compact={true}
      icon={source ? <SourceLogo id={source} size={LOGO_PX} /> : <LogIn />}
      title={site ? t`Sign in to ${{ source: site }}` : t`Sign in to see this`}
      action={
        <Button variant="contained" onClick={() => openSettings('accounts')}>
          {t`Sign in…`}
        </Button>
      }
    >
      {children ?? t`The site shares this only with a signed-in account.`}
    </EmptyState>
  )
}
