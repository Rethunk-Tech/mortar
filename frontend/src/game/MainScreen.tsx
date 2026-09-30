import { Box } from '@mui/material'
import { useEffect } from 'react'
import { LoaderBanner } from '../loader/Banner.tsx'
import { useProfiles } from '../profiles/store.ts'
import { QueueSheet } from '../queue/QueueSheet.tsx'
import { Detail } from './Detail.tsx'
import { Sidebar } from './Sidebar.tsx'

export function MainScreen({ game }: { game: string }) {
  const load = useProfiles((s) => s.load)
  useEffect(() => {
    load(game)
  }, [game, load])
  return (
    <Box sx={{ position: 'relative', height: '100%', display: 'flex', flexDirection: 'column' }}>
      <LoaderBanner game={game} />
      <Box sx={{ flex: 1, minHeight: 0, display: 'flex' }}>
        <Sidebar game={game} />
        <Detail />
      </Box>
      <QueueSheet />
    </Box>
  )
}
