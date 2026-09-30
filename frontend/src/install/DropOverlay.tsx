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
const TITLE_BAR = 36

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
        top: TITLE_BAR,
        left: 0,
        right: 0,
        bottom: 0,
        zIndex: theme.zIndex.modal,
        pointerEvents: 'none',
        bgcolor: 'rgba(0,0,0,0.30)',
      })}
    >
      <Box
        sx={(theme) => ({
          position: 'absolute',
          inset: 12,
          border: `3px dashed ${theme.palette.primary.main}`,
          borderRadius: '16px',
          bgcolor: alpha(theme.palette.primary.main, TINT_ALPHA),
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          gap: 2,
          textAlign: 'center',
          p: 3,
        })}
      >
        <Box sx={{ color: 'primary.main', display: 'flex' }}>
          <Download size={80} strokeWidth={1.8} />
        </Box>
        {ready ? (
          <>
            <Typography sx={{ fontSize: 38, fontWeight: 700 }}>
              {t`Drop to install into ${profile.name}`}
            </Typography>
            <Typography sx={{ maxWidth: 640, fontSize: 15, lineHeight: 1.5 }}>
              {t`Zip, RAR and 7z. Mortar reads the mods inside and shows you what was added.`}
            </Typography>
          </>
        ) : (
          <Typography sx={{ fontSize: 38, fontWeight: 700 }}>
            {t`Open a profile to install mods`}
          </Typography>
        )}
      </Box>
    </Box>
  )
}
