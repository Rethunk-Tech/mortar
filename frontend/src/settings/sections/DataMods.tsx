import { useLingui } from '@lingui/react/macro'
import Box from '@mui/material/Box'
import { useEffect, useState } from 'react'
import type { CacheInfo as CacheInfoValue } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/models.ts'
import {
  CacheInfo,
  ClearCache,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/service.ts'
import { formatBytes } from '../../i18n/bytes.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'

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
  const [info, setInfo] = useState<CacheInfoValue | null>(null)
  useEffect(() => {
    if (!open) {
      return
    }
    CacheInfo().then(setInfo).catch(reportUnexpected)
  }, [open])
  const size = info === null ? '' : formatBytes(info.size)
  const path = info?.path ?? ''
  return (
    <ConfirmDialog
      open={open}
      title={t`Clear cache?`}
      body={
        <>
          {t`Removes cached Nexus and SMAPI details and problem scans. Use it if an API key was revoked or the details look wrong. Mortar fetches them again as needed, so the next problem check takes longer.`}
          {info === null ? null : (
            <Box component="span" sx={{ display: 'block', mt: 1, wordBreak: 'break-all' }}>
              {t`${size} in ${path}`}
            </Box>
          )}
        </>
      }
      confirmLabel={t`Clear cache`}
      onCancel={onClose}
      onConfirm={() => {
        ClearCache()
          .then(() => {
            onClose()
            onCleared()
            useToasts.getState().push({ kind: 'success', title: t`Cache cleared` })
          })
          .catch(reportUnexpected)
      }}
    />
  )
}

export { CacheClearDialog }
