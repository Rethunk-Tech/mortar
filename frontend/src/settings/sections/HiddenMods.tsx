import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import {
  UnhideBrowseAuthor,
  UnhideBrowseMod,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useCurrentGame } from '../../nav/currentGame.ts'
import { space } from '../../theme/density.ts'
import { saveSetting } from '../persist.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { useSettings } from '../store.ts'

function HiddenMods() {
  const { t } = useLingui()
  const game = useCurrentGame()
  const hidden = useSettings((s) => s.games?.[game]?.hiddenMods) ?? []
  const authors = useSettings((s) => s.games?.[game]?.hiddenAuthors) ?? []
  if (hidden.length === 0 && authors.length === 0) {
    return null
  }
  return (
    <SettingRow
      label={t`Hidden mods`}
      description={t`Left out of Browse. Links and imports still install them.`}
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
        {hidden.map((h) => (
          <Box
            key={`${h.source}:${h.id}`}
            sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}
          >
            <Typography sx={{ minWidth: 160 }}>{h.name || h.id}</Typography>
            <Button
              size="small"
              onClick={() => saveSetting(() => UnhideBrowseMod(game, h.source, h.id))}
            >
              {t`Unhide`}
            </Button>
          </Box>
        ))}
        {authors.map((a) => (
          <Box key={`author:${a}`} sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
            <Typography sx={{ minWidth: 160 }}>{t`Everything by ${a}`}</Typography>
            <Button size="small" onClick={() => saveSetting(() => UnhideBrowseAuthor(game, a))}>
              {t`Unhide`}
            </Button>
          </Box>
        ))}
      </Box>
    </SettingRow>
  )
}

export { HiddenMods }
