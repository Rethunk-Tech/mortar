import { useLingui } from '@lingui/react/macro'
import { Box, Button, CircularProgress } from '@mui/material'
import { Plus, RotateCcw } from 'lucide-react'
import { useState } from 'react'
import { useProfiles } from '../profiles/store.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { NewProfileDialog } from './NewProfileDialog.tsx'

export function ProfilesFailed() {
  const { t } = useLingui()
  const failed = useProfiles((s) => s.failed)
  const load = useProfiles((s) => s.load)
  return (
    <EmptyState
      icon={<RotateCcw />}
      title={t`Could not read your profiles`}
      action={
        <Button
          variant="contained"
          startIcon={<RotateCcw size={16} />}
          onClick={() => {
            load(failed).catch(reportUnexpected)
          }}
        >
          {t`Retry`}
        </Button>
      }
    >
      {t`Mortar could not load the profiles for this game.`}
    </EmptyState>
  )
}

export function ProfilesLoading() {
  return (
    <Box sx={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <CircularProgress />
    </Box>
  )
}

export function ProfilesEmpty() {
  const { t } = useLingui()
  const [creating, setCreating] = useState(false)
  return (
    <EmptyState
      icon={<Plus />}
      title={t`No profiles yet`}
      action={
        <Button
          variant="contained"
          startIcon={<Plus size={16} />}
          onClick={() => setCreating(true)}
        >
          {t`Create your first profile`}
        </Button>
      }
    >
      {t`A profile holds one set of mods for this game.`}
      <NewProfileDialog open={creating} onClose={() => setCreating(false)} />
    </EmptyState>
  )
}
