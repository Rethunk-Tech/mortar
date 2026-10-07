import { useLingui } from '@lingui/react/macro'
import { RefreshCw } from 'lucide-react'
import { PageActions } from '../game/PageActions.tsx'
import { checkForUpdates } from '../shell/checkForUpdates.ts'
import { IconAction } from '../shell/IconAction.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { AddArchive } from './Toolbar.tsx'

// The Mods tab's buttons in the tab header: the update check (F5) and the Add split button.
export function ModsHeaderActions() {
  const { t } = useLingui()
  return (
    <PageActions>
      <IconAction
        label={t`Check for updates (F5)`}
        icon={<RefreshCw size={16} aria-hidden={true} />}
        onClick={() => {
          checkForUpdates().catch(reportUnexpected)
        }}
      />
      <AddArchive variant="outlined" toolbar={true} />
    </PageActions>
  )
}
