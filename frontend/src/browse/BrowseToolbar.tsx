import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import { PrefSegmented } from '../settings/PrefControls.tsx'
import { OfflineGate } from '../shell/OfflineGate.tsx'
import { useOfflineReason } from '../shell/offlineText.ts'
import { SearchField } from '../shell/SearchField.tsx'
import { ViewToggle } from '../shell/ViewToggle.tsx'
import { ALL } from './browseConstants.ts'
import { useBrowseView } from './view.ts'

function BrowseToolbar({
  sources,
  source,
  onSource,
  draft,
  onDraft,
  placeholder,
}: {
  sources: { value: string; label: string }[]
  source: string
  onSource: (next: string) => void
  draft: string
  onDraft: (next: string) => void
  placeholder: string
}) {
  const { t } = useLingui()
  const view = useBrowseView((s) => s.view)
  const setView = useBrowseView((s) => s.setView)
  const offline = useOfflineReason(source === ALL ? [] : [source])
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, px: 2, pt: 1.25, pb: 0.75 }}>
      <ViewToggle value={view} onChange={setView} />
      <PrefSegmented value={source} label={t`Source`} options={sources} onChange={onSource} />
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <OfflineGate reason={offline}>
          <SearchField
            autoFocus={true}
            value={draft}
            onChange={onDraft}
            label={placeholder}
            fullWidth={true}
          />
        </OfflineGate>
      </Box>
    </Box>
  )
}

export { BrowseToolbar }
