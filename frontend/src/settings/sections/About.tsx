import { useLingui } from '@lingui/react/macro'
import { Box, Link } from '@mui/material'
import { Browser } from '@wailsio/runtime'
import { Logo } from '../../brand/Logo.tsx'
import { reportUnexpected } from '../../toasts/report.ts'

const LICENCE = 'https://github.com/Rethunk-AI/mortar/blob/main/LICENSE'

const body = { color: 'rgba(235,235,240,0.95)' }

export function About() {
  const { t } = useLingui()
  return (
    <Box
      sx={{ display: 'flex', flexDirection: 'column', gap: '12px', fontSize: 14, lineHeight: 1.55 }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
        <Logo size={40} />
        <Box sx={{ display: 'flex', flexDirection: 'column' }}>
          <Box component="span" sx={{ fontSize: 18, fontWeight: 700 }}>
            {t`Mortar`}
          </Box>
          <Box component="span" sx={{ color: 'rgba(225,225,230,0.95)' }}>
            {t`AGPL-3.0 · Rethunk-AI/mortar`}
          </Box>
        </Box>
      </Box>
      <Box sx={{ fontWeight: 600 }}>{t`Built on`}</Box>
      <Box sx={body}>
        {t`SMAPI by Pathoschild (LGPL-3.0) · the Stardew mod dataset by Pathoschild (CC-BY-SA 4.0 / MIT) · Wails · and open-source libraries under their own licences.`}{' '}
        <Link
          component="button"
          onClick={() => Browser.OpenURL(LICENCE).catch(reportUnexpected)}
          sx={{ fontSize: 'inherit', verticalAlign: 'baseline' }}
        >
          {t`Mortar's licence`}
        </Link>
      </Box>
      <Box sx={{ fontWeight: 600 }}>{t`Default background`}</Box>
      <Box sx={body}>
        {t`Fedora 44 default wallpaper (f44-01-night) by the Fedora Design Team, CC-BY-SA-4.0.`}
      </Box>
      <Box sx={{ fontWeight: 600 }}>{t`Not affiliated`}</Box>
      <Box sx={body}>
        {t`Stardew Valley is ConcernedApe's. Game art shown in Mortar is read from your own Steam install. Nexus Mods and GitHub content belongs to its authors.`}
      </Box>
    </Box>
  )
}
