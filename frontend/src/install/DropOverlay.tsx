import { useLingui } from '@lingui/react/macro'
import { alpha, Box, Typography } from '@mui/material'
import { Events } from '@wailsio/runtime'
import { Download } from 'lucide-react'
import { useEffect, useState } from 'react'
import { EnableRequirementsDialog } from '../mods/EnableRequirementsDialog.tsx'
import { useLocked } from '../mods/useLocked.ts'
import { routeGame, useNav } from '../nav/store.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { openImport } from '../share/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { splitDropped } from './dropped.ts'
import { MissingDepsDialog } from './MissingDepsDialog.tsx'
import { RemapDialog } from './RemapDialog.tsx'
import { useInstall } from './store.ts'

// Wails toggles this class on the element carrying data-file-drop-target while a file drag is over it.
const ACTIVE_CLASS = 'file-drop-target-active'
const TINT_ALPHA = 0.08

export function DropOverlay({ target }: { target: HTMLElement | null }) {
  const { t } = useLingui()
  const [dragging, setDragging] = useState(false)
  const inGame = useNav((s) => routeGame(s.route) !== null)
  const locked = useLocked()
  const profile = useProfiles(openProfileOf)
  const ready = inGame && profile !== undefined && !locked

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
        const { archives, mortar } = splitDropped(event.data ?? [])
        if (archives.length > 0) {
          useInstall.getState().install(archives).catch(reportUnexpected)
        }
        if (mortar) {
          const onGame = routeGame(useNav.getState().route) !== null
          openImport({ profileId: onGame ? useProfiles.getState().openId : '', file: mortar })
        }
      }),
    [],
  )

  return (
    <>
      <MissingDepsDialog />
      <EnableRequirementsDialog />
      <RemapDialog />
      {dragging ? (
        <Box
          sx={(theme) => ({
            position: 'absolute',
            top: 'var(--title-bar)',
            left: 0,
            right: 0,
            bottom: 0,
            zIndex: theme.zIndex.modal,
            pointerEvents: 'none',
            bgcolor: 'var(--mortar-overlay-30)',
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
                {locked ? t`Stop the game to change mods` : t`Open a profile to install mods`}
              </Typography>
            )}
          </Box>
        </Box>
      ) : null}
    </>
  )
}
