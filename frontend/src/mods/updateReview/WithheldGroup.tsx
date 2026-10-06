import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import type { Update } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { Mod } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { sameId } from '../lookup.ts'
import { ModNameLink } from '../ModNameLink.tsx'
import { openPage } from '../menu.ts'
import { useMods } from '../store.ts'
import { Version } from './Version.tsx'

/** Updates SMAPI reports that Mortar cannot install because their Nexus page is hidden or removed, with the reason. */
export function WithheldGroup({ withheld, mods }: { withheld: Update[]; mods: Mod[] }) {
  const { t } = useLingui()
  const setSkipVersion = useMods((s) => s.setSkipVersion)
  if (withheld.length === 0) {
    return null
  }
  return (
    <Box>
      <Typography sx={{ px: 3, pt: 1.5, fontWeight: 600 }}>
        {t`Not offered (${withheld.length})`}
      </Typography>
      <Typography sx={{ px: 3, fontSize: 13, color: 'text.secondary' }}>
        {t`Nexus hides or has removed these pages, so there is no download to offer.`}
      </Typography>
      <Box role="list">
        {withheld.map((u) => {
          const mod = mods.find((m) => m.key === u.key && sameId(m.id, u.id))
          return (
            <Box
              key={`${u.key}/${u.id}`}
              role="listitem"
              sx={{ display: 'flex', alignItems: 'center', gap: 1.5, px: 3, py: 1 }}
            >
              <Box sx={{ flex: 1, minWidth: 0, display: 'flex' }}>
                <ModNameLink id={u.id} modKey={u.key} name={u.name} />
              </Box>
              <Version>{u.installed}</Version>
              <Version isNew={true}>{u.version}</Version>
              {u.url ? <Button onClick={() => openPage(u.url)}>{t`Open on Nexus`}</Button> : null}
              {mod ? (
                <Button onClick={() => setSkipVersion(mod, u.version).catch(reportUnexpected)}>
                  {t`Skip this version`}
                </Button>
              ) : null}
            </Box>
          )
        })}
      </Box>
    </Box>
  )
}
