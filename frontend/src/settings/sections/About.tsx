import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import { Logo } from '../../brand/Logo.tsx'

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
        {t`SMAPI by Pathoschild (LGPL-3.0) · the Stardew mod dataset by Pathoschild (CC-BY-SA 4.0 / MIT) · Wails · open-source libraries listed in Licences.`}
      </Box>
      <Box sx={{ fontWeight: 600 }}>{t`Not affiliated`}</Box>
      <Box sx={body}>
        {t`Stardew Valley is ConcernedApe's. Game art shown in Mortar is read from your own Steam install. Nexus Mods and GitHub content belongs to its authors.`}
      </Box>
    </Box>
  )
}
