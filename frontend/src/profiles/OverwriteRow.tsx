import { useLingui } from '@lingui/react/macro'
import { Box, Button, DialogContent, Typography } from '@mui/material'
import { useCallback, useEffect, useState } from 'react'
import {
  ClearOverwrite,
  OpenOverwrite,
  OverwriteFiles,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { reportUnexpected, toastError } from '../toasts/report.ts'

// What the game wrote outside its settings folders for this profile (BepInEx games keep it in the profile's overwrite
// folder). Nothing shows when there is none.
export function OverwriteRow({
  game,
  profileId,
  open,
}: {
  game: string
  profileId: string
  open: boolean
}) {
  const { t } = useLingui()
  const [files, setFiles] = useState(0)
  const [asking, setAsking] = useState(false)
  const load = useCallback(() => {
    OverwriteFiles(game, profileId).then(setFiles).catch(reportUnexpected)
  }, [game, profileId])
  useEffect(() => {
    if (open && game !== '') {
      load()
    }
  }, [open, game, load])
  if (files === 0) {
    return null
  }
  return (
    <DialogContent sx={{ pt: 0 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <Typography sx={{ flex: 1 }}>{t`Overwrite (${files} files)`}</Typography>
        <Button onClick={() => OpenOverwrite(game, profileId).catch(reportUnexpected)}>
          {t`Open folder`}
        </Button>
        <Button color="error" onClick={() => setAsking(true)}>
          {t`Clear`}
        </Button>
      </Box>
      <ConfirmDialog
        open={asking}
        title={t`Clear the overwrite folder?`}
        body={t`This deletes the ${files} files the game wrote for this profile outside its settings folders.`}
        confirmLabel={t`Clear`}
        color="error"
        onCancel={() => setAsking(false)}
        onConfirm={() => {
          setAsking(false)
          ClearOverwrite(game, profileId)
            .then(load)
            .catch((error: unknown) => toastError(t`Could not clear the overwrite folder`, error))
        }}
      />
    </DialogContent>
  )
}
