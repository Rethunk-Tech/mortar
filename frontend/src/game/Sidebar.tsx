import { Box } from '@mui/material'
import { PlayControl } from '../launch/PlayControl.tsx'
import { compact } from './compact.ts'

const WIDTH = 220
const RAIL = 56

export function Sidebar({ game }: { game: string }) {
  return (
    <Box
      component="nav"
      sx={{
        width: WIDTH,
        flexShrink: 0,
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'flex-end',
        borderRight: '2px solid',
        borderColor: 'primary.main',
        bgcolor: 'background.paper',
        [compact]: { width: RAIL },
      }}
    >
      <PlayControl game={game} />
    </Box>
  )
}
