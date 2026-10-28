import { useLingui } from '@lingui/react/macro'
import { Box, Button, Link, Typography } from '@mui/material'
import { useState } from 'react'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { OfflineGate } from '../shell/OfflineGate.tsx'
import { updateSources, useOfflineReason } from '../shell/offlineText.ts'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { nexusIdOf, sameId, visibleUpdates } from './lookup.ts'
import { openPage } from './menu.ts'
import { useNexusEntry } from './nexusDetails.ts'
import { offersNexusDownload } from './nexusMark.ts'
import { accent } from './paper.ts'
import { useMods } from './store.ts'
import { EverywhereDialog } from './updateReview/EverywhereDialog.tsx'
import { installedCaution } from './updateReview/wants.ts'
import { useUpdates } from './updates.ts'

export function UpdateBanner({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const profile = useProfiles(openProfileOf)
  const nexusId = profile ? nexusIdOf(profile, mod) : 0
  const game = useProfiles((s) => s.game?.id ?? '')
  const details = useNexusEntry(game, nexusId)?.details
  const raw = useUpdates((s) => s.updates)
  const setReviewing = useUpdates((s) => s.setReviewing)
  const [everywhere, setEverywhere] = useState(false)
  const setSkipVersion = useMods((s) => s.setSkipVersion)
  const mine = visibleUpdates(raw, profile).filter((u) => u.key === mod.key && sameId(u.id, mod.id))
  const unofficial = mine.find((u) => u.unofficial)
  const update = mine.find((u) => !u.unofficial)
  const blocked =
    Boolean(update) &&
    (update?.nexusId ?? 0) > 0 &&
    !offersNexusDownload(details?.page.status, details?.page.available)
  const official = blocked ? undefined : update
  const offline = useOfflineReason(update ? updateSources(update) : [])
  if (!(official || unofficial)) {
    return null
  }
  return (
    <Box
      sx={{
        px: space.pad,
        py: 0.75,
        display: 'flex',
        flexDirection: 'column',
        gap: 0.75,
        borderRadius: '6px',
        bgcolor: accent.fill,
        border: '1px solid',
        borderColor: accent.line,
      }}
    >
      <Box sx={{ minWidth: 0, display: 'flex', flexDirection: 'column', gap: 0.25 }}>
        {official ? (
          <Typography sx={{ fontSize: 'base.fontSize' }}>
            {t`Update available: ${{ installed: official.installed }} to ${{ version: official.version }}`}
          </Typography>
        ) : null}
        {unofficial ? (
          <Typography sx={{ fontSize: 'base.fontSize' }}>
            {t`Unofficial update available: ${{ version: unofficial.version }}`}
            {unofficial.url ? (
              <>
                {' '}
                <Link
                  component="button"
                  onClick={() => openPage(unofficial.url)}
                  sx={{ fontSize: 'base.fontSize' }}
                >
                  {t`Open page`}
                </Link>
              </>
            ) : null}
          </Typography>
        ) : null}
      </Box>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: space.gap }}>
        <OfflineGate reason={offline}>
          <Button size="small" variant="contained" onClick={() => setReviewing(true)}>
            {t`Update`}
          </Button>
        </OfflineGate>
        {official ? (
          <Button
            size="small"
            variant="outlined"
            onClick={() => setSkipVersion(mod, official.version).catch(reportUnexpected)}
          >
            {t`Skip this update`}
          </Button>
        ) : null}
        {update && installedCaution(update) === '' ? (
          <OfflineGate reason={offline}>
            <Button size="small" variant="outlined" onClick={() => setEverywhere(true)}>
              {t`Update in all profiles that have it`}
            </Button>
          </OfflineGate>
        ) : null}
      </Box>
      <EverywhereDialog
        open={everywhere}
        game={game}
        mods={[{ id: mod.id, newKey: 'latest' }]}
        onClose={() => setEverywhere(false)}
      />
    </Box>
  )
}
