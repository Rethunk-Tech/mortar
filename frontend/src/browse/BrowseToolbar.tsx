import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Tooltip } from '@mui/material'
import { Layers } from 'lucide-react'
import type { ReactNode } from 'react'
import { SourceLogo } from '../brand/sources/SourceLogo.tsx'
import { hasSourceLogo } from '../brand/sources/sourceIcons.ts'
import { OfflineGate } from '../shell/OfflineGate.tsx'
import { useOfflineReason } from '../shell/offlineText.ts'
import { SearchField } from '../shell/SearchField.tsx'
import { ViewToggle } from '../shell/ViewToggle.tsx'
import { space } from '../theme/density.ts'
import { ALL } from './browseConstants.ts'
import { useBrowseView } from './view.ts'

interface SourceOption {
  value: string
  label: string
  unavailable?: string | undefined
}

const sourceButton = (active: boolean) => ({
  minWidth: 34,
  height: `calc(${space.control} - 6px)`,
  px: 1,
  borderRadius: '6px',
  fontSize: 13,
  bgcolor: active ? 'var(--mortar-hairline-16)' : 'transparent',
  color: active ? 'var(--mortar-ink)' : 'text.secondary',
  '&:hover': { bgcolor: active ? 'var(--mortar-hairline-16)' : 'var(--mortar-hairline-muted)' },
  '&.Mui-disabled': { opacity: 0.4 },
})

// The source picker shows each site's logo, named in its tooltip; a source with no logo shows its name.
function SourceToggle({
  sources,
  value,
  onChange,
}: {
  sources: SourceOption[]
  value: string
  onChange: (next: string) => void
}) {
  const { t } = useLingui()
  return (
    <Box
      role="group"
      aria-label={t`Source`}
      sx={{
        display: 'flex',
        p: '3px',
        gap: '2px',
        bgcolor: 'var(--mortar-overlay-30)',
        borderRadius: '8px',
        flexShrink: 0,
      }}
    >
      {sources.map((o) => {
        let mark: ReactNode = o.label
        if (o.value === ALL) {
          mark = <Layers size={15} />
        } else if (hasSourceLogo(o.value)) {
          mark = <SourceLogo id={o.value} size={16} />
        }
        return (
          <Tooltip key={o.value} title={o.unavailable ?? o.label} describeChild={true}>
            <span>
              <ButtonBase
                aria-label={o.label}
                aria-pressed={o.value === value}
                disabled={Boolean(o.unavailable)}
                onClick={() => onChange(o.value)}
                sx={sourceButton(o.value === value)}
              >
                {mark}
              </ButtonBase>
            </span>
          </Tooltip>
        )
      })}
    </Box>
  )
}

function BrowseToolbar({
  sources,
  source,
  onSource,
  draft,
  onDraft,
  placeholder,
}: {
  sources: SourceOption[]
  source: string
  onSource: (next: string) => void
  draft: string
  onDraft: (next: string) => void
  placeholder: string
}) {
  const view = useBrowseView((s) => s.view)
  const setView = useBrowseView((s) => s.setView)
  const offline = useOfflineReason(source === ALL ? [] : [source])
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, px: 2, pt: 1.25, pb: 0.75 }}>
      <ViewToggle value={view} onChange={setView} />
      <SourceToggle sources={sources} value={source} onChange={onSource} />
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
