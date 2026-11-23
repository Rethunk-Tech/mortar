import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import { useState } from 'react'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { layoutQuestion, stopAsking, useLayoutAsk } from './unknownLayout.ts'

const SHOWN_FILES = 40

/** The confirm behind "Install anyway" for an archive no layout takes: the files are what the player judges by. */
export function UnknownLayoutDialog() {
  const { t } = useLingui()
  const asking = useLayoutAsk((s) => s.asking)
  const close = useLayoutAsk((s) => s.close)
  const [always, setAlways] = useState(false)
  const files = asking?.layout.files ?? []
  return (
    <ConfirmationDialog
      open={asking !== null}
      tone="warning"
      title={t`Install ${{ name: asking?.title ?? '' }} anyway?`}
      body={`${layoutQuestion(asking?.layout.reason ?? '', true)} ${t`Installing it places the files as the archive has them, so check the mod's page before you play.`}`}
      confirmLabel={t`Install anyway`}
      checkbox={{ label: t`Don't ask again for this game`, checked: always, onChange: setAlways }}
      onCancel={() => {
        setAlways(false)
        close()
      }}
      onConfirm={() => {
        const current = asking
        const remember = always
        setAlways(false)
        close()
        if (!current) {
          return
        }
        Promise.resolve()
          .then(() => (remember ? stopAsking(current.layout.game) : undefined))
          .then(() => current.confirm())
          .catch(reportUnexpected)
      }}
    >
      <Box component="ul" sx={{ m: 0, pl: 3, maxHeight: 240, overflow: 'auto' }}>
        {files.slice(0, SHOWN_FILES).map((f) => (
          <Typography key={f} component="li" variant="body2" sx={{ wordBreak: 'break-all' }}>
            {f}
          </Typography>
        ))}
        {files.length > SHOWN_FILES ? (
          <Typography component="li" variant="body2">
            {t`and ${{ count: files.length - SHOWN_FILES }} more`}
          </Typography>
        ) : null}
      </Box>
    </ConfirmationDialog>
  )
}
