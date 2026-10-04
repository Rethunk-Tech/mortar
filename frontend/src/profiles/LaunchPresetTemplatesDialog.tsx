import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Typography,
} from '@mui/material'
import { useCallback, useEffect, useState } from 'react'
import type { LaunchPresetTemplate } from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import {
  ListLaunchPresetTemplates,
  RemoveLaunchPresetTemplate,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { reportUnexpected } from '../toasts/report.ts'

/** Game-wide launch preset templates: Add copies one into the profile; the copy is independent. */
export function LaunchPresetTemplatesDialog({
  open,
  gameId,
  onClose,
  onAdd,
}: {
  open: boolean
  gameId: string
  onClose: () => void
  onAdd: (template: LaunchPresetTemplate) => void
}) {
  const { t } = useLingui()
  const [templates, setTemplates] = useState<LaunchPresetTemplate[]>([])
  const [removing, setRemoving] = useState<LaunchPresetTemplate | null>(null)
  const reload = useCallback(() => {
    ListLaunchPresetTemplates(gameId)
      .then((next) => setTemplates(next ?? []))
      .catch(reportUnexpected)
  }, [gameId])
  useEffect(() => {
    if (open) {
      reload()
    }
  }, [open, reload])
  return (
    <Dialog open={open} onClose={onClose} fullWidth={true} maxWidth="sm">
      <DialogTitle>{t`Launch preset templates`}</DialogTitle>
      <DialogContent>
        <Typography sx={{ fontSize: 12, color: 'text.secondary', mb: 1 }}>
          {t`Templates are shared by every profile of this game. Adding one copies it into this profile's launch presets.`}
        </Typography>
        {templates.length === 0 ? (
          <Typography sx={{ color: 'text.secondary' }}>
            {t`No templates yet. Use Save as template on a launch preset.`}
          </Typography>
        ) : (
          templates.map((tpl) => (
            <Box key={tpl.name} sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 0.5 }}>
              <Typography noWrap={true} sx={{ flex: 1, minWidth: 0 }}>
                {tpl.name}
              </Typography>
              <Button onClick={() => onAdd(tpl)}>{t`Add`}</Button>
              <Button color="error" onClick={() => setRemoving(tpl)}>{t`Delete`}</Button>
            </Box>
          ))
        )}
      </DialogContent>
      <DialogActions>
        <Button variant="contained" onClick={onClose}>{t`Close`}</Button>
      </DialogActions>
      <ConfirmDialog
        open={removing !== null}
        title={t`Delete ${removing?.name ?? ''}?`}
        body={t`This template will be deleted. Presets already copied into profiles stay.`}
        confirmLabel={t`Delete template`}
        color="error"
        onCancel={() => setRemoving(null)}
        onConfirm={() => {
          const gone = removing
          setRemoving(null)
          if (gone) {
            RemoveLaunchPresetTemplate(gameId, gone.name ?? '')
              .then(reload)
              .catch(reportUnexpected)
          }
        }}
      />
    </Dialog>
  )
}
