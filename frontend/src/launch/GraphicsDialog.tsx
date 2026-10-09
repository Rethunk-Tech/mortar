import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Link,
  Typography,
} from '@mui/material'
import { useGameInfo } from '../games/info.ts'
import { openPage } from '../mods/menu.ts'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { graphicsOptions } from './graphicsAsk.ts'
import { cancelling } from './playModeState.ts'
import { useLaunch } from './store.ts'

export function GraphicsDialog() {
  const { t } = useLingui()
  const ask = useLaunch((s) => s.graphicsAsk)
  const cancel = useLaunch((s) => s.dismissGraphicsAsk)
  const answer = useLaunch((s) => s.answerGraphics)
  const quit = cancelling(cancel)
  const info = useGameInfo(ask?.game)
  const graphics = info?.graphics
  const name = info?.name ?? ''
  return (
    <Dialog open={ask !== null} onClose={quit} slotProps={{ paper: { sx: { maxWidth: 480 } } }}>
      <DialogTitle>{t`Choose a graphics API for ${name}`}</DialogTitle>
      <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
        {ask?.afterEarlyExit ? (
          <DialogContentText sx={{ fontWeight: 600 }}>
            {t`The game closed right after starting last time.`}
          </DialogContentText>
        ) : null}
        <DialogContentText>{graphics?.explanation}</DialogContentText>
        {graphics ? (
          <Link
            component="button"
            variant="body2"
            sx={{ textAlign: 'left' }}
            onClick={() => openPage(graphics.source)}
          >
            {graphics.reason}
          </Link>
        ) : null}
        <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
          {t`You can change this later in ${name}'s settings, under Play.`}
        </Typography>
      </DialogContent>
      <DialogActions sx={{ flexWrap: 'wrap', gap: space.gap }}>
        <Button onClick={quit}>{t`Cancel`}</Button>
        {graphics
          ? graphicsOptions(graphics)
              .reverse()
              .map((o) => (
                <Button
                  key={o.id}
                  variant={o.recommended ? 'contained' : 'text'}
                  onClick={() => answer(o.id).catch(reportUnexpected)}
                >
                  {o.recommended ? t`${o.label} (Recommended)` : o.label}
                </Button>
              ))
          : null}
      </DialogActions>
    </Dialog>
  )
}
