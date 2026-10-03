import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import type { Update } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { useNexus } from '../../settings/nexus.ts'
import { Fold } from '../../shell/Fold.tsx'
import { reportUnexpected } from '../../toasts/report.ts'
import { changelogNoteIsRisky, changelogsBetween } from '../changelogRange.ts'
import { loadDetails, useNexusDetails } from '../nexusDetails.ts'

export function Changes({ update }: { update: Update }) {
  const { t } = useLingui()
  const details = useNexusDetails((s) => s.byId[update.nexusId]?.details)
  const signedIn = useNexus((s) => s.signedIn)
  if (!update.nexusId) {
    return null
  }
  if (!details) {
    return (
      <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
        {signedIn ? t`Changelog is not cached yet.` : t`Sign in to Nexus to load the changelog.`}
        {signedIn ? (
          <Button
            size="small"
            onClick={() => loadDetails(update.nexusId).catch(reportUnexpected)}
            sx={{ ml: 0.75, minWidth: 0, p: 0, fontSize: 12, textTransform: 'none' }}
          >
            {t`Load changes`}
          </Button>
        ) : null}
      </Typography>
    )
  }
  const all = details.changelogs ?? []
  const logs = changelogsBetween(all, update.installed, update.version)
  if (all.length === 0) {
    return (
      <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
        {t`This mod has no changelog.`}
      </Typography>
    )
  }
  if (logs.length === 0) {
    return (
      <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
        {t`No changelog entries between these versions.`}
      </Typography>
    )
  }
  return (
    <Fold
      title={t`${plural(logs.length, { one: '# version of changes', other: '# versions of changes' })}`}
    >
      {logs.map((c) => (
        <Box key={c.version}>
          <Typography sx={{ fontSize: 13, fontWeight: 700 }}>{c.version}</Typography>
          {(c.notes ?? []).map((n) => (
            <Typography
              key={n}
              sx={{
                fontSize: 13,
                pl: 2,
                whiteSpace: 'pre-line',
                overflowWrap: 'anywhere',
                color: changelogNoteIsRisky(n) ? 'warning.main' : undefined,
              }}
            >
              {`• ${n}`}
            </Typography>
          ))}
        </Box>
      ))}
    </Fold>
  )
}
