import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Tooltip } from '@mui/material'
import { Layers } from 'lucide-react'
import type { ReactNode } from 'react'
import { SourceLogo } from '../brand/sources/SourceLogo.tsx'
import { hasSourceLogo } from '../brand/sources/sourceIcons.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
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
  px: space.gap,
  borderRadius: '6px',
  fontSize: 13,
  bgcolor: active ? 'var(--mortar-hairline-16)' : 'transparent',
  color: active ? 'var(--mortar-ink)' : 'text.secondary',
  '&:hover': { bgcolor: active ? 'var(--mortar-hairline-16)' : 'var(--mortar-hairline-muted)' },
  '&.Mui-disabled': { opacity: 0.4 },
})

// The source picker shows each site's logo, named in its tooltip; a source with no logo shows its name.
// One source in the picker. A source that cannot be used says why to keyboard users too; one that can only names
// itself.
function SourceButton({
  option,
  active,
  onChange,
}: {
  option: SourceOption
  active: boolean
  onChange: (next: string) => void
}) {
  let mark: ReactNode = option.label
  if (option.value === ALL) {
    mark = <Layers size={15} />
  } else if (hasSourceLogo(option.value)) {
    mark = <SourceLogo id={option.value} size={16} />
  }
  const button = (
    <ButtonBase
      aria-label={option.label}
      aria-pressed={active}
      disabled={Boolean(option.unavailable)}
      onClick={() => onChange(option.value)}
      sx={sourceButton(active)}
    >
      {mark}
    </ButtonBase>
  )
  return option.unavailable ? (
    <DisabledReason title={option.unavailable} disabled={true}>
      {button}
    </DisabledReason>
  ) : (
    <Tooltip title={option.label}>{button}</Tooltip>
  )
}

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
      {sources.map((o) => (
        <SourceButton key={o.value} option={o} active={o.value === value} onChange={onChange} />
      ))}
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
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: space.gap,
        px: space.gutter,
        py: space.gap,
      }}
    >
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
