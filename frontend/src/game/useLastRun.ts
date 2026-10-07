import { useEffect, useState } from 'react'
import { Runs } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { profileCardLastPlayedIso } from '../games/profileCardLastPlayed.ts'
import { useSettings } from '../settings/store.ts'

// When the profile's latest run started ('' when it never ran); re-read after each run, which moves `updated`.
export function useLastRun(game: string, profileId: string, updated: string): string {
  const played = useSettings((s) => s.lastPlayed?.[game])
  const [started, setStarted] = useState('')
  const key = `${game}\0${profileId}\0${updated}`
  useEffect(() => {
    const [g = '', id = ''] = key.split('\0')
    let live = true
    Runs(g, id)
      .then((runs) => live && setStarted(runs?.[0]?.started ?? ''))
      .catch(() => live && setStarted(''))
    return () => {
      live = false
    }
  }, [key])
  return profileCardLastPlayedIso(profileId, played, { [profileId]: started })
}
