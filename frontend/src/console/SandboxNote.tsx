import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Tooltip, Typography } from '@mui/material'
import { useLoaded } from '@rethunk/react-load-hooks'
import { SandboxReport } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { calloutFill, calloutLine } from '../theme/callout.ts'
import { space } from '../theme/density.ts'

const NONE = { writable: [], denied: [] }
const SHOWN = 5

// Says a run was sandboxed and, when its logs show the game or a mod reaching for a folder the sandbox hides, which.
export function SandboxNote({
  game,
  profile,
  run,
}: {
  game: string
  profile: string
  run: string
}) {
  const { t } = useLingui()
  const { data } = useLoaded(
    () =>
      SandboxReport(game, profile, run).then(
        (r) => ({ writable: r.writable ?? [], denied: r.denied ?? [] }),
        () => NONE,
      ),
    [game, profile, run],
    NONE,
  )
  if (data.writable.length === 0) {
    return null
  }
  const denied = data.denied
  return (
    <Box
      sx={{
        mx: space.gutter,
        mb: space.gap,
        display: 'flex',
        flexDirection: 'column',
        gap: space.gap,
      }}
    >
      <Tooltip title={<Box sx={{ whiteSpace: 'pre-line' }}>{data.writable.join('\n')}</Box>}>
        <Typography sx={{ fontSize: 'base.fontSize', color: 'text.secondary' }}>
          {plural(data.writable.length, {
            one: 'This run was sandboxed: the game could write # folder',
            other: 'This run was sandboxed: the game could write # folders',
          })}
        </Typography>
      </Tooltip>
      {denied.length > 0 ? (
        <Box
          sx={{
            px: space.pad,
            py: space.gap,
            bgcolor: calloutFill('warning'),
            border: '1px solid',
            borderColor: calloutLine('warning'),
            borderRadius: '6px',
          }}
        >
          <Typography sx={{ fontSize: 'base.fontSize', fontWeight: 700 }}>
            {t`The sandbox hid folders from the game or a mod`}
          </Typography>
          <Typography sx={{ fontSize: 'base.fontSize' }}>
            {t`If a mod needs one of these, turn off “Sandbox the game” for this game or profile on the Play page.`}
          </Typography>
          <Box
            component="ul"
            sx={{ m: 0, pl: 3, fontFamily: 'monospace', fontSize: 'base.fontSize' }}
          >
            {denied.slice(0, SHOWN).map((p) => (
              <li key={p}>{p}</li>
            ))}
          </Box>
          {denied.length > SHOWN ? (
            <Typography sx={{ fontSize: 'base.fontSize', color: 'text.secondary' }}>
              {plural(denied.length - SHOWN, { one: 'and # more', other: 'and # more' })}
            </Typography>
          ) : null}
        </Box>
      ) : null}
    </Box>
  )
}
