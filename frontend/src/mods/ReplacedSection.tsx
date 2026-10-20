import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { Info } from 'lucide-react'
import { useState } from 'react'
import {
  DeleteReplaced,
  RestoreReplaced,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/service.ts'
import type { ReplacedFile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useMods } from './store.ts'
import { openTarget } from './storeView.ts'

const rowSx = {
  display: 'flex',
  alignItems: 'flex-start',
  gap: space.gap,
  flexShrink: 0,
  pl: space.pad,
  pr: 0.75,
  py: space.gap,
  bgcolor: 'var(--mortar-overlay-45)',
  borderRadius: '6px',
} as const

function ReplacedRow({ file, onRestore }: { file: ReplacedFile; onRestore: () => void }) {
  const { t } = useLingui()
  return (
    <Box sx={rowSx}>
      <Box sx={{ display: 'flex', color: 'text.secondary', mt: '2px' }}>
        <Info size={16} aria-hidden={true} />
      </Box>
      <Typography
        title={file.path}
        sx={{ flex: 1, minWidth: 0, fontSize: 14, whiteSpace: 'normal', wordBreak: 'break-word' }}
      >
        {file.path}
      </Typography>
      <Button size="small" aria-label={t`Restore ${{ path: file.path }}`} onClick={onRestore}>
        {t`Restore`}
      </Button>
    </Box>
  )
}

// The files sync moved aside because a package's own file belongs at their path: an older install left them there.
export function ReplacedSection({ files }: { files: ReplacedFile[] }) {
  const { t } = useLingui()
  const [confirming, setConfirming] = useState(false)
  const reload = () => useMods.getState().loadProblems()
  const run = (work: (game: string, id: string) => Promise<void>) => {
    const at = openTarget()
    if (at) {
      work(at.game, at.id).then(reload).catch(reportUnexpected)
    }
  }
  if (files.length === 0) {
    return null
  }
  return (
    <Box>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, mb: 1 }}>
        <Typography sx={{ flex: 1, fontSize: 14 }}>
          {t`${plural(files.length, { one: 'Replaced # old file Mortar did not install', other: 'Replaced # old files Mortar did not install' })}`}
        </Typography>
        <Button size="small" color="error" sx={{ height: 26 }} onClick={() => setConfirming(true)}>
          {t`Delete all`}
        </Button>
      </Box>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
        {files.map((file) => (
          <ReplacedRow
            key={file.path}
            file={file}
            onRestore={() => run((game, id) => RestoreReplaced(game, id, file.path))}
          />
        ))}
      </Box>
      <ConfirmDialog
        open={confirming}
        title={t`Delete the replaced files?`}
        body={t`Mortar replaced these with the mods' own files. Deleting them cannot be undone.`}
        confirmLabel={t`Delete all`}
        color="error"
        onCancel={() => setConfirming(false)}
        onConfirm={() => {
          setConfirming(false)
          run((game, id) => DeleteReplaced(game, id))
        }}
      />
    </Box>
  )
}
