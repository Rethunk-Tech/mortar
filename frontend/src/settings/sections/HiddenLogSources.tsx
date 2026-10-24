import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import type { HiddenLogSource } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { ShowLogSource } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { sameSource, useHiddenSources } from '../../console/hiddenSources.ts'
import { useCurrentGame } from '../../nav/currentGame.ts'
import { space } from '../../theme/density.ts'
import { persist } from '../persist.ts'
import { Searchable, SettingsSection } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

const ENGINE = 'unity:'
// A mod's message by its log column (launch.ModMessagePrefix), as SMAPI names the mod writing each line.
const MOD = 'mod:'

function whereFrom(source: string, engine: string): string {
  if (source.startsWith(ENGINE)) {
    return engine
  }
  return source.startsWith(MOD) ? source.slice(MOD.length) : source.replace('::', '.')
}

// Hidden sources grouped by where their lines come from (a method, a mod, or the engine), groups and lines in name order.
function grouped(hidden: readonly HiddenLogSource[], engine: string) {
  const groups = new Map<string, HiddenLogSource[]>()
  for (const h of hidden) {
    const from = whereFrom(h.source, engine)
    groups.set(from, [...(groups.get(from) ?? []), h])
  }
  const shown = (h: HiddenLogSource) => h.text || h.label || h.source
  return [...groups.entries()]
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([from, items]) => ({
      from,
      items: items.sort((a, b) => shown(a).localeCompare(shown(b))),
    }))
}

/** The messages hidden from the game's Console with right-click, for every profile of the game, by source. */
export function HiddenLogSources() {
  const { t } = useLingui()
  const game = useCurrentGame()
  const { hidden, defaults } = useHiddenSources(game)
  const own = useSettings((s) => s.games?.[game]?.hiddenLogSources) ?? []
  const isDefault = (h: HiddenLogSource) =>
    defaults.some((d) => sameSource(d, h)) && !own.some((o) => sameSource(o, h))
  const engine = t`Engine`
  return (
    <Searchable terms={t`Hidden messages`}>
      <SettingsSection
        title={t`Hidden messages`}
        description={t`Right-click a line in the Console to hide every line from where it came from, in every profile of this game. The game stops writing them where its bridge can. Messages Mortar hides by default for this game are listed too.`}
      >
        {hidden.length === 0 ? (
          <Typography sx={{ p: space.pad, color: 'text.secondary' }}>{t`None`}</Typography>
        ) : null}
        {grouped(hidden, engine).map(({ from, items }) => (
          <Box
            key={from}
            sx={{ p: space.pad, display: 'flex', flexDirection: 'column', gap: space.gap }}
          >
            <Typography sx={{ fontWeight: 600, overflowWrap: 'anywhere' }}>{from}</Typography>
            {items.map((h) => (
              <Box
                key={`${h.text}\t${h.object}`}
                sx={{ display: 'flex', alignItems: 'flex-start', gap: space.gap, minWidth: 0 }}
              >
                <Box sx={{ flex: 1, minWidth: 0 }}>
                  <Typography
                    title={h.label}
                    sx={{
                      fontFamily: 'monospace',
                      whiteSpace: 'pre-wrap',
                      overflowWrap: 'anywhere',
                    }}
                  >
                    {h.text || t`Every message`}
                  </Typography>
                  <Typography
                    variant="body2"
                    color="text.secondary"
                    sx={{ overflowWrap: 'anywhere' }}
                  >
                    {h.object
                      ? t`Only lines about ${{ object: h.object }}`
                      : t`Lines about every object`}
                    {isDefault(h) ? ` · ${t`Hidden by default`}` : null}
                  </Typography>
                </Box>
                <Button
                  size="small"
                  sx={{ flexShrink: 0 }}
                  aria-label={t`Show ${{ name: h.label || h.source }} again`}
                  onClick={() =>
                    persist(
                      () => ShowLogSource(game, h.source, h.text ?? '', h.object ?? ''),
                      t`Could not save that setting`,
                    )
                  }
                >
                  {t`Show again`}
                </Button>
              </Box>
            ))}
          </Box>
        ))}
      </SettingsSection>
    </Searchable>
  )
}
