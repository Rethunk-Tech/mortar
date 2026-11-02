import type { ComponentProps } from 'react'
import { PageLinkButton } from '../PageLinkButton.tsx'
import { TextInput } from './FormDialog.tsx'

/** A secret token field: masked, never offered to a password manager. The hint is a link to the page that issues one
 * unless the caller supplies its own. */
export function TokenInput({
  page,
  pageLabel,
  hint,
  ...rest
}: Omit<ComponentProps<typeof TextInput>, 'type' | 'autoComplete' | 'hint'> & {
  page?: string
  pageLabel?: string
  hint?: ComponentProps<typeof TextInput>['hint']
}) {
  return (
    <TextInput
      {...rest}
      type="password"
      autoComplete="off"
      hint={
        hint ??
        (page !== undefined && pageLabel !== undefined ? (
          <PageLinkButton url={page} sx={{ px: 0 }}>
            {pageLabel}
          </PageLinkButton>
        ) : undefined)
      }
    />
  )
}
