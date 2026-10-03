import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { Download, Upload } from 'lucide-react'
import { ClearCache } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/datasvc/service.ts'
import type { ImportPreview } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import {
  ExportSettings,
  PreviewImportSettings,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { reportUnexpected } from '../../toasts/report.ts'
import { nowrap } from './dataStyles.ts'

function CacheClearDialog({
  open,
  onClose,
  onCleared,
}: {
  open: boolean
  onClose: () => void
  onCleared: () => void
}) {
  const { t } = useLingui()
  return (
    <ConfirmDialog
      open={open}
      title={t`Clear cache`}
      body={t`Removes cached Nexus and SMAPI details and problem scans. Use it if an API key was revoked or the details look wrong. Mortar fetches them again as needed, so the next problem check takes longer.`}
      confirmLabel={t`Clear`}
      onCancel={onClose}
      onConfirm={() => {
        ClearCache()
          .then(() => {
            onClose()
            onCleared()
          })
          .catch(reportUnexpected)
      }}
    />
  )
}

function DataSettingsFiles({ onImport }: { onImport: (p: ImportPreview) => void }) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
      <Button
        onClick={() => ExportSettings().catch(reportUnexpected)}
        startIcon={<Download size={16} />}
        sx={{ alignSelf: 'flex-start', ...nowrap }}
      >
        {t`Export settings…`}
      </Button>
      <Button
        onClick={() => {
          PreviewImportSettings()
            .then((next) => {
              if (next.raw) {
                onImport(next)
              }
            })
            .catch(reportUnexpected)
        }}
        startIcon={<Upload size={16} />}
        sx={{ alignSelf: 'flex-start', ...nowrap }}
      >
        {t`Import settings…`}
      </Button>
    </Box>
  )
}

export { CacheClearDialog, DataSettingsFiles }
