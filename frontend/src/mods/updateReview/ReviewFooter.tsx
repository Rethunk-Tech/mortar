import { useLingui } from '@lingui/react/macro'
import { Box, Button, Checkbox, DialogActions, FormControlLabel, Typography } from '@mui/material'
import { ShieldCheck } from 'lucide-react'

export function ReviewFooter({
  wantedCount,
  signedIn,
  uncachedIds,
  loadingAll,
  propagateAll,
  onClose,
  onLoadAll,
  onPropagate,
  onUpdate,
  onEverywhere,
}: {
  wantedCount: number
  signedIn: boolean
  uncachedIds: number[]
  loadingAll: boolean
  propagateAll: boolean
  onClose: () => void
  onLoadAll: () => void
  onPropagate: (on: boolean) => void
  onUpdate: () => void
  onEverywhere: () => void
}) {
  const { t } = useLingui()
  return (
    <DialogActions
      sx={{
        px: 3,
        py: 2,
        gap: 1.5,
        flexDirection: 'column',
        alignItems: 'stretch',
        bgcolor: 'var(--mortar-overlay-20)',
        '& > :not(style) ~ :not(style)': { ml: 0 },
      }}
    >
      <Box sx={{ display: 'flex', gap: 1.25, alignItems: 'flex-start' }}>
        <Box sx={{ color: 'text.secondary', display: 'flex', pt: '2px' }}>
          <ShieldCheck size={18} aria-hidden={true} />
        </Box>
        <Typography sx={{ fontSize: 13, lineHeight: 1.45, color: 'text.secondary' }}>
          {t`Update downloads a mod's new file from Nexus or GitHub. For other pages, download the archive and drop it on this window: Mortar updates the mod in place, keeps its settings, and backs up your saves first. Roll back any mod later from its details.`}{' '}
          {t`Mortar checks for updates at startup, when you press F5, and at most once an hour while it is running.`}
        </Typography>
      </Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
        {wantedCount > 0 ? (
          <FormControlLabel
            control={<Checkbox checked={propagateAll} onChange={(_, on) => onPropagate(on)} />}
            label={t`Also update other profiles that hold the same files`}
            sx={{ mr: 'auto', '& .MuiFormControlLabel-label': { fontSize: 13 } }}
          />
        ) : (
          <Box sx={{ mr: 'auto' }} />
        )}
        {signedIn && uncachedIds.length > 0 ? (
          <Button
            variant="text"
            disabled={loadingAll}
            onClick={onLoadAll}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {loadingAll ? t`Loading changes…` : t`Load all changes`}
          </Button>
        ) : null}
        <Button variant="text" color="inherit" onClick={onClose} sx={{ whiteSpace: 'nowrap' }}>
          {t`Close`}
        </Button>
        {wantedCount > 0 ? (
          <Button
            variant="outlined"
            color="inherit"
            onClick={onEverywhere}
            sx={{ whiteSpace: 'nowrap' }}
          >
            {t`Update all everywhere…`}
          </Button>
        ) : null}
        {wantedCount > 0 ? (
          <Button variant="contained" onClick={onUpdate} sx={{ whiteSpace: 'nowrap' }}>
            {t`Update ${wantedCount}`}
          </Button>
        ) : null}
      </Box>
    </DialogActions>
  )
}
