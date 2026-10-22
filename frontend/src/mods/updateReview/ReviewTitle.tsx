import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, DialogTitle, Typography } from '@mui/material'
import { space } from '../../theme/density.ts'

export function ReviewTitle({
  count,
  profileName,
  checkedLabel,
  canLoadAll,
  loadingAll,
  onLoadAll,
}: {
  count: number
  profileName: string
  checkedLabel: string
  canLoadAll: boolean
  loadingAll: boolean
  onLoadAll: () => void
}) {
  const { t } = useLingui()
  return (
    <DialogTitle component="div" sx={{ p: space.pad }}>
      <Typography component="h2" sx={{ fontSize: 22, fontWeight: 700 }}>
        {plural(count, {
          one: `# update for ${{ profile: profileName }}`,
          other: `# updates for ${{ profile: profileName }}`,
        })}
      </Typography>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, minHeight: 30 }}>
        <Typography sx={{ fontSize: 13 }}>{checkedLabel}</Typography>
        {canLoadAll ? (
          <Button size="small" disabled={loadingAll} onClick={onLoadAll}>
            {loadingAll ? t`Loading changes…` : t`Load all changes`}
          </Button>
        ) : null}
      </Box>
    </DialogTitle>
  )
}
