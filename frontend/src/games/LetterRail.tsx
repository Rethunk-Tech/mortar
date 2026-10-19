import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase } from '@mui/material'
import { memo } from 'react'
import { useRoomy } from '../theme/roomy.ts'

const RAIL_LETTER_MAX_PX = 48
const RAIL_ROOMY_FONT_PX = 11
const RAIL_ROOMY_WIDTH_PX = 44
const RAIL_WIDTH_PX = 28

// Memoised on its letters and the current one: the list renders again as it scrolls, and the rail's
// twenty-odd buttons need not.
export const LetterRail = memo(
  function LetterRailView({
    letters,
    current,
    onJump,
  }: {
    letters: string[]
    current: string
    onJump: (letter: string) => void
  }) {
    const { t } = useLingui()
    // Touch targets are 44px in roomy mode, the theme's rule for every button, so the rail widens to hold them.
    const roomy = useRoomy((s) => s.roomy)
    return (
      <Box
        component="nav"
        aria-label={t`Jump to a letter`}
        sx={{
          width: roomy ? RAIL_ROOMY_WIDTH_PX : RAIL_WIDTH_PX,
          flexShrink: 0,
          py: '8px',
          display: 'flex',
          flexDirection: 'column',
          bgcolor: 'background.paper',
          // The letters share the rail's height and their type follows it, so a tall window does not leave a small
          // cluster in the middle; centred when capped, scrolling from the first letter when they do not fit.
          containerType: 'size',
          overflowY: 'auto',
          scrollbarWidth: 'none',
          '&::-webkit-scrollbar': { display: 'none' },
          '& > :first-of-type': { mt: 'auto' },
          '& > :last-child': { mb: 'auto' },
        }}
      >
        {letters.map((letter) => (
          <ButtonBase
            key={letter}
            aria-label={t`Games starting with ${letter}`}
            aria-current={letter === current ? 'true' : undefined}
            onClick={() => onJump(letter)}
            sx={{
              flex: roomy ? '0 0 auto' : '1 1 0',
              minHeight: 12,
              maxHeight: RAIL_LETTER_MAX_PX,
              fontFamily: 'inherit',
              fontSize: roomy ? RAIL_ROOMY_FONT_PX : 'clamp(11px, 1.6cqh, 15px)',
              fontWeight: 700,
              color: letter === current ? 'var(--mortar-accent-ink)' : 'var(--mortar-ink-sec)',
              '&:hover, &:focus-visible': { color: 'text.primary' },
            }}
          >
            {letter}
          </ButtonBase>
        ))}
      </Box>
    )
  },
  (a, b) =>
    a.current === b.current && a.onJump === b.onJump && a.letters.join('') === b.letters.join(''),
)
