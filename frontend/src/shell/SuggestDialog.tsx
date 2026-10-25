import { useLingui } from '@lingui/react/macro'
import { Button, Typography } from '@mui/material'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { openPage } from '../mods/menu.ts'
import { type Destination, GO, getDestination, SUGGEST_KEY, setDestination } from './destination.ts'
import { AppDialog, DialogFooter } from './dialog/AppDialog.tsx'
import { useSuggest } from './suggest.ts'

const ROUTES: Record<Destination, string> = {
  discord: `${GO}suggestions/`,
  github: 'https://github.com/Rethunk-Tech/mortar/issues/new?template=feature_request.yml',
}

export function SuggestDialog() {
  const { t } = useLingui()
  const open = useSuggest((s) => s.open)
  const close = () => useSuggest.setState({ open: false })
  // Read on each open, so the button that was last used is the primary one.
  const last = open ? getDestination(SUGGEST_KEY) : 'github'
  const go = (to: Destination) => () => {
    setDestination(SUGGEST_KEY, to)
    close()
    openPage(ROUTES[to])
  }
  const variant = (to: Destination) => (last === to ? 'contained' : 'outlined')
  const order: Destination[] = last === 'github' ? ['discord', 'github'] : ['github', 'discord']
  return (
    <AppDialog
      open={open}
      onClose={close}
      title={t`Suggest a feature`}
      size="sm"
      footer={
        <DialogFooter>
          <Button onClick={close}>{t`Cancel`}</Button>
          {order.map((to) => (
            <Button key={to} variant={variant(to)} onClick={go(to)}>
              {sourceLabel(to)}
            </Button>
          ))}
        </DialogFooter>
      }
    >
      <Typography>
        {t`Post your idea where you prefer. ${{ app: sourceLabel('discord') }} lets others vote on it; ${{ source: sourceLabel('github') }} keeps it next to the code.`}
      </Typography>
    </AppDialog>
  )
}
