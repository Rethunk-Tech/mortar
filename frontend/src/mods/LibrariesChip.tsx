import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, Chip, Popover, Typography } from '@mui/material'
import { useState } from 'react'
import { space } from '../theme/density.ts'
import { useDetail } from './detail.ts'
import type { ListRow } from './listColumns.ts'
import { modId } from './lookup.ts'
import { ModMenu } from './ModMenu.tsx'
import { ModSwitch } from './parts.tsx'

// The implicit mods a card view folds into the card of the mod that needs them: a chip that opens them with their
// switch and menu.
export function LibrariesChip({ libraries }: { libraries: readonly ListRow[] | undefined }) {
  const { i18n } = useLingui()
  const openDetail = useDetail((s) => s.show)
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  if (!libraries || libraries.length === 0) {
    return null
  }
  const label = i18n._(plural(libraries.length, { one: '+# library', other: '+# libraries' }))
  return (
    <>
      <Chip
        size="small"
        label={label}
        aria-haspopup="true"
        onClick={(e) => setAnchor(e.currentTarget)}
        sx={{
          height: 20,
          fontSize: 11,
          bgcolor: 'action.selected',
          color: 'text.secondary',
          '& .MuiChip-label': { px: 0.75 },
        }}
      />
      <Popover
        open={anchor !== null}
        anchorEl={anchor}
        onClose={() => setAnchor(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
      >
        <Box sx={{ p: space.pad, minWidth: 280, maxWidth: 420 }}>
          {libraries.map((r) => (
            <Box key={modId(r.mod)} sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
              <ButtonBase
                data-mod-id={modId(r.mod)}
                onClick={() => {
                  openDetail(r.mod)
                  setAnchor(null)
                }}
                sx={{
                  flex: 1,
                  minWidth: 0,
                  justifyContent: 'flex-start',
                  textAlign: 'left',
                  pl: (r.depth ?? 1) > 1 ? 2 : 0,
                }}
              >
                <Typography noWrap={true} title={r.mod.name} sx={{ fontSize: 14 }}>
                  {r.mod.name}
                </Typography>
              </ButtonBase>
              <ModSwitch mod={r.mod} />
              <ModMenu mod={r.mod} />
            </Box>
          ))}
        </Box>
      </Popover>
    </>
  )
}
