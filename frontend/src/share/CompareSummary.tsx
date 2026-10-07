import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Typography } from '@mui/material'
import type {
  Mod,
  Preview,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/models.ts'
import { Fold } from '../shell/Fold.tsx'
import { space } from '../theme/density.ts'

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
  const rows: { label: string; items: string[] }[] = [
    { label: t`Missing in ${targetName}`, items: names(missing) },
    { label: t`Different version`, items: names(different) },
    { label: t`Only in ${{ name: targetName }}`, items: onlyYours },
  ]
  return (
    <Box
      sx={{
        px: space.gap,
        pt: 0.5,
        pb: space.gap,
        display: 'flex',
        flexDirection: 'column',
        gap: 0.75,
      }}
    >
      <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
        {t`${plural(matching.length, { one: '# mod already matches', other: '# mods already match' })} ${targetName}.`}
      </Typography>
      {rows
        .filter((r) => r.items.length > 0)
        .map((r) => (
          <Fold key={r.label} title={`${r.label} (${r.items.length})`}>
            <Typography
              tabIndex={0}
              sx={{ fontSize: 13, maxHeight: 120, overflowY: 'auto', mt: 0.5 }}
            >
              {r.items.join(', ')}
            </Typography>
          </Fold>
        ))}
    </Box>
  )
}
