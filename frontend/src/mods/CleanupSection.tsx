import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { type ComponentProps, useState } from 'react'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useMods } from './store.ts'

function CleanupRow({
  cleanup,
}: {
  cleanup: { key: string; uniqueId: string; name: string; reason?: string }
}) {
  const { t } = useLingui()
  const remove = useMods((s) => s.remove)
  const mod = useMods((s) => s.mods.find((candidate) => candidate.key === cleanup.key))
  const who = cleanup.name.trim() === '' ? t`Unknown mod` : cleanup.name
  const reason = cleanup.reason || 'Not needed by any enabled mod'
  return (
    <Box
      role="alert"
      sx={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: 1.25,
        flexShrink: 0,
        pl: 1.5,
        pr: 0.75,
        py: 1,
        fontSize: 14,
        bgcolor: 'var(--mortar-overlay-45)',
        borderRadius: '6px',
      }}
    >
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography
          title={cleanup.name.trim() === '' ? cleanup.uniqueId : undefined}
          sx={{ fontSize: 14, whiteSpace: 'normal', wordBreak: 'break-word' }}
        >
          {t`${who}: ${reason}`}
        </Typography>
      </Box>
      <DisabledReason title={t`This mod is no longer in the profile.`} disabled={mod === undefined}>
        <Button
          size="small"
          disabled={mod === undefined}
          onClick={() => {
            if (mod !== undefined) {
              remove(mod).catch(reportUnexpected)
            }
          }}
        >
          {t`Remove`}
        </Button>
      </DisabledReason>
    </Box>
  )
}

function removeCleanup(
  cleanup: { key: string }[],
  removeMany: ReturnType<typeof useMods.getState>['removeMany'],
) {
  const mods = cleanup
    .map((item) => useMods.getState().mods.find((mod) => mod.key === item.key))
    .filter((mod): mod is NonNullable<typeof mod> => mod !== undefined)
  removeMany(mods).catch(reportUnexpected)
}

export function CleanupSection({
  cleanup,
}: {
  cleanup: ComponentProps<typeof CleanupRow>['cleanup'][]
}) {
  const { t } = useLingui()
  const removeMany = useMods((s) => s.removeMany)
  const [confirmCleanup, setConfirmCleanup] = useState(false)
  if (cleanup.length === 0) {
    return null
  }
  return (
    <Box>
      <Box sx={{ mb: 1, display: 'flex', alignItems: 'center' }}>
        <Typography sx={{ fontSize: 13, fontWeight: 600, color: 'text.secondary' }}>
          {t`Cleanup`}
        </Typography>
        <Button size="small" sx={{ ml: 1, height: 26 }} onClick={() => setConfirmCleanup(true)}>
          {t`Remove all`}
        </Button>
      </Box>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
        {cleanup.map((item) => (
          <CleanupRow key={item.key} cleanup={item} />
        ))}
      </Box>
      <ConfirmDialog
        open={confirmCleanup}
        title={t`Remove all ${cleanup.length} mods from this profile?`}
        body={t`This change can be undone from History.`}
        confirmLabel={t`Remove all`}
        color="error"
        onCancel={() => setConfirmCleanup(false)}
        onConfirm={() => {
          setConfirmCleanup(false)
          removeCleanup(cleanup, removeMany)
        }}
      />
    </Box>
  )
}
