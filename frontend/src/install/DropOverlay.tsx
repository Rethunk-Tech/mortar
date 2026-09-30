import { useLingui } from '@lingui/react/macro'
import { alpha, Box, Typography } from '@mui/material'
import { Events } from '@wailsio/runtime'
import { Download } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useNav } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useInstall } from './store.ts'

// Wails toggles this class on the element carrying data-file-drop-target while a file drag is over it.
const ACTIVE_CLASS = 'file-drop-target-active'
const TINT_ALPHA = 0.08

export function DropOverlay({ target }: { target: HTMLElement | null }) {
  const { t } = useLingui()
  const [dragging, setDragging] = useState(false)
  const inGame = useNav((s) => s.route.name === 'game')
  const profile = useProfiles((s) => s.profiles.find((p) => p.id === s.openId))
  const ready = inGame && profile !== undefined

  useEffect(() => {
    if (!target) {
      return
    }
    const sync = () => setDragging(target.classList.contains(ACTIVE_CLASS))
    const observer = new MutationObserver(sync)
    observer.observe(target, { attributes: true, attributeFilter: ['class'] })
    return () => observer.disconnect()
  }, [target])

  useEffect(
    () =>
      Events.On('files:dropped', (event) => {
        useInstall
          .getState()
          .install(event.data ?? [])
          .catch(reportUnexpected)
      }),
    [],
  )

  if (!dragging) {
    return null
  }
  return (
    <Box
      sx={(theme) => ({
        position: 'absolute',
        top: 48,
        left: 12,
        right: 12,
        bottom: 12,
        zIndex: theme.zIndex.modal,
        pointerEvents: 'none',
        border: `2px dashed ${theme.palette.primary.main}`,
        borderRadius: '10px',
        bgcolor: alpha(theme.palette.primary.main, TINT_ALPHA),
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 1.5,
        textAlign: 'center',
        p: 3,
      })}
    >
      <Download size={64} />
      {ready ? (
        <>
          <Typography sx={{ fontSize: 24, fontWeight: 600 }}>
            {t`Drop to install into ${profile.name}`}
          </Typography>
          <Typography sx={{ color: 'text.secondary' }}>
            {t`Zip, RAR and 7z. Mortar reads the mods inside and shows you what was added.`}
          </Typography>
        </>
      ) : (
        <Typography sx={{ fontSize: 24, fontWeight: 600 }}>
          {t`Open a profile to install mods`}
        </Typography>
      )}
    </Box>
  )
}
