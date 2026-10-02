import { useLingui } from '@lingui/react/macro'
import { Button, Dialog, DialogContent, DialogTitle, ListItemIcon, MenuItem } from '@mui/material'
import { Download } from 'lucide-react'
import type { SourceInfo } from '../../bindings/github.com/Rethunk-AI/mortar/internal/migrate/models.ts'
import { ExternalPreview } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { openImport } from '../share/store.ts'
import { reportUnexpected } from '../toasts/report.ts'

export function ExternalImportMenuItems({
  sources,
  onPick,
}: {
  sources: SourceInfo[]
  onPick: (source: SourceInfo) => void
}) {
  const { t } = useLingui()
  return sources.map((item) => (
    <MenuItem
      key={item.kind}
      onClick={() => {
        onPick(item)
      }}
    >
      <ListItemIcon sx={{ color: 'inherit' }}>
        <Download size={16} aria-hidden={true} />
      </ListItemIcon>
      {t`From ${item.name}…`}
    </MenuItem>
  ))
}

export function ExternalImportProfileDialog({
  game,
  source,
  onClose,
}: {
  game: string
  source: SourceInfo | null
  onClose: () => void
}) {
  const { t } = useLingui()
  return (
    <Dialog open={source !== null} onClose={onClose}>
      <DialogTitle>{t`Choose a profile to import`}</DialogTitle>
      <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, minWidth: 360 }}>
        {(source?.profiles ?? []).map((profile) => (
          <Button
            key={profile.id}
            onClick={() => {
              if (!source) {
                return
              }
              ExternalPreview(game, source.kind, profile.id)
                .then((preview) => {
                  openImport({ external: preview })
                  onClose()
                })
                .catch(reportUnexpected)
            }}
            sx={{ justifyContent: 'flex-start', textTransform: 'none' }}
          >
            {profile.name}
          </Button>
        ))}
      </DialogContent>
    </Dialog>
  )
}
