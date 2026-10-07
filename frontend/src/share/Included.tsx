import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { TriangleAlert } from 'lucide-react'
import { useState } from 'react'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { useProfiles } from '../profiles/store.ts'
import { IncludeOptions } from './IncludeOptions.tsx'
import type { ShownInfo } from './logic.ts'
import { includedKeys, leftOutCounts } from './methods.ts'
import { offersFomod, type ShareInclude } from './shareDefaults.ts'
import { useShareDialog } from './store.ts'

const SHOWN_NAMES = 30

function ModNames({ info }: { info: ShownInfo }) {
  const { t } = useLingui()
  const names = info.groups.flatMap((g) =>
    g.mods.map((name) => ({ key: `${g.source}-${name}`, name, source: g.source })),
  )
  const shown = names.slice(0, SHOWN_NAMES)
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75 }}>
        {info.groups.map((g) => {
          const { count } = g
          return (
            <Box
              key={g.source}
              component="span"
              sx={{
                px: 1,
                py: '2px',
                borderRadius: '10px',
                bgcolor: 'var(--mortar-hairline-muted)',
                fontSize: 12,
              }}
            >
              {t`${count} ${{ source: sourceLabel(g.source) }}`}
            </Box>
          )
        })}
      </Box>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75 }}>
        {shown.map((m) => (
          <Box
            key={m.key}
            component="span"
            sx={{
              px: 1.25,
              py: 0.5,
              borderRadius: '4px',
              bgcolor: 'var(--mortar-card-hover)',
              fontSize: 13,
              whiteSpace: 'nowrap',
            }}
          >
            {m.name}
          </Box>
        ))}
        {names.length > shown.length ? (
          <Box component="span" sx={{ px: 1.25, py: 0.5, fontSize: 13, color: 'text.secondary' }}>
            {t`and ${{ count: names.length - shown.length }} more`}
          </Box>
        ) : null}
      </Box>
    </Box>
  )
}

// What the share holds, the settings behind it (Change), and the mods it leaves out (Which?).
export function Included({
  info,
  include,
  onInclude,
  file,
}: {
  info: ShownInfo
  include: ShareInclude
  onInclude: (next: ShareInclude) => void
  file: boolean
}) {
  const { t, i18n } = useLingui()
  const [changing, setChanging] = useState(false)
  const [which, setWhich] = useState(false)
  const [showMods, setShowMods] = useState(false)
  const profileId = useShareDialog((s) => s.profileId)
  const fomod = useProfiles((s) =>
    offersFomod(s.profiles.find((p) => p.id === profileId)?.entries, s.game?.sources),
  )
  const labels = {
    notes: t`Notes`.toLocaleLowerCase(i18n.locale),
    fomodChoices: t`FOMOD choices`,
    configFiles: t`Config files`.toLocaleLowerCase(i18n.locale),
    disabledMods: '',
  }
  const { count } = info
  const mods = include.disabledMods
    ? plural(count, { one: '# mod', other: '# mods' })
    : plural(count, { one: '# enabled mod', other: '# enabled mods' })
  const line = [mods, ...includedKeys(include, { file, fomod }).map((k) => labels[k])].join(' · ')
  const { local, other } = leftOutCounts(info.leftOut)
  const reason = (id: string) => {
    switch (id) {
      case 'local':
        return t`Added from an archive, so it only exists on this computer.`
      case 'off':
        return t`Disabled in this profile`
      default:
        return t`Its source is not known.`
    }
  }
  const archive = file
    ? plural(local, {
        one: "# mod you added from an archive can't go in the file",
        other: "# mods you added from archives can't go in the file",
      })
    : plural(local, {
        one: "# mod you added from an archive can't be linked",
        other: "# mods you added from archives can't be linked",
      })
  const rest = plural(other, { one: '# more mod is left out', other: '# more mods are left out' })
  const warning = [local > 0 ? archive : '', other > 0 ? rest : ''].filter(Boolean).join(' · ')
  const link = { minWidth: 0, p: 0, ml: 1, fontSize: 14 }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.25 }}>
      <Typography sx={{ fontWeight: 600 }}>{t`Included`}</Typography>
      <Box sx={{ display: 'flex', alignItems: 'center' }}>
        <Typography sx={{ fontSize: 14 }}>{line}</Typography>
        <Button
          variant="text"
          sx={link}
          aria-expanded={changing}
          onClick={() => setChanging(!changing)}
        >
          {t`Change`}
        </Button>
        <Button
          variant="text"
          sx={link}
          aria-expanded={showMods}
          onClick={() => setShowMods(!showMods)}
        >
          {t`Show mods`}
        </Button>
      </Box>
      {showMods ? <ModNames info={info} /> : null}
      {changing ? <IncludeOptions value={include} onChange={onInclude} file={file} /> : null}
      {info.leftOut.length > 0 ? (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Box sx={{ display: 'flex', color: 'warning.main' }}>
            <TriangleAlert size={16} />
          </Box>
          <Typography sx={{ fontSize: 14 }}>{warning}</Typography>
          <Button variant="text" sx={link} aria-expanded={which} onClick={() => setWhich(!which)}>
            {t`Which?`}
          </Button>
        </Box>
      ) : null}
      {which ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.75 }}>
          {info.leftOut.map((o) => (
            <Box
              key={`${o.name}-${o.reason}`}
              sx={{ display: 'flex', alignItems: 'center', gap: 1, fontSize: 13 }}
            >
              <Box
                component="span"
                sx={{ px: 1, py: '2px', borderRadius: '4px', bgcolor: 'var(--mortar-overlay-30)' }}
              >
                {o.name}
              </Box>
              <Box component="span" sx={{ color: 'text.secondary' }}>
                {reason(o.reason)}
              </Box>
            </Box>
          ))}
        </Box>
      ) : null}
    </Box>
  )
}
