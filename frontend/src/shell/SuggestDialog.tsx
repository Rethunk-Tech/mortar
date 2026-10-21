import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from '@mui/material'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { openPage } from '../mods/menu.ts'
import { type Destination, GO, getDestination, SUGGEST_KEY, setDestination } from './destination.ts'
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
  return (
    <Dialog open={open} onClose={close} fullWidth={true} maxWidth="xs">
      <DialogTitle>{t`Suggest a feature`}</DialogTitle>
      <DialogContent>
        <Typography>
          {t`Post your idea where you prefer. Discord lets others vote on it; GitHub keeps it next to the code.`}
        </Typography>
      </DialogContent>
      <DialogActions>
        <Button onClick={close}>{t`Cancel`}</Button>
        <Button variant={variant('github')} onClick={go('github')}>
          {sourceLabel('github')}
        </Button>
        <Button variant={variant('discord')} onClick={go('discord')}>
          {sourceLabel('discord')}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
