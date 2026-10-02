import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import type {
  Mod,
  Preview,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/sharesvc/models.ts'

// CompareSummary is how a shared profile lines up with the profile it would join, so friends can match mods before
// playing together: what already matches, what this import would download, what differs in version, and what only
// the user's profile has (which Replace removes and Add keeps).
export function CompareSummary({ preview, targetName }: { preview: Preview; targetName: string }) {
  const { t } = useLingui()
  const mods = preview.mods ?? []
  const names = (list: Mod[]) => list.map((m) => m.name)
  const missing = mods.filter((m) => m.state === 'download' || m.state === 'dependency')
  const different = mods.filter((m) => m.different)
  const matching = mods.filter((m) => m.state === 'installed' && !m.different)
  const onlyYours = preview.replace?.remove ?? []
  const rows: { label: string; items: string[]; tone: string }[] = [
    { label: t`Missing in ${targetName}`, items: names(missing), tone: 'warning.main' },
    { label: t`Different version`, items: names(different), tone: 'warning.main' },
    { label: t`Only in ${targetName}`, items: onlyYours, tone: 'text.secondary' },
  ]
  return (
    <Box sx={{ px: 1, pt: 0.5, pb: 1, display: 'flex', flexDirection: 'column', gap: 0.75 }}>
      <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
        {t`${matching.length} mods already match ${targetName}.`}
      </Typography>
      {rows
        .filter((r) => r.items.length > 0)
        .map((r) => (
          <Typography key={r.label} sx={{ fontSize: 13 }}>
            <Box component="span" sx={{ color: r.tone, fontWeight: 600 }}>
              {`${r.label} (${r.items.length}): `}
            </Box>
            {r.items.join(', ')}
          </Typography>
        ))}
    </Box>
  )
}
