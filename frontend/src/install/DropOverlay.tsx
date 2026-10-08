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
import { space } from '../theme/density.ts'
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
    // Wails can miss the end of a drag (it ignores a dragleave out of the window, and a drag cancelled outside
    // never reaches it), leaving the class set. The OS holds the pointer and keyboard during a drag, so any of
    // these in the window means the drag is over.
    const end = () => {
      if (target.classList.contains(ACTIVE_CLASS)) {
        target.classList.remove(ACTIVE_CLASS)
      }
    }
    const events = ['pointermove', 'pointerdown', 'keydown'] as const
    for (const e of events) {
      globalThis.addEventListener(e, end)
    }
    return () => {
      observer.disconnect()
      for (const e of events) {
        globalThis.removeEventListener(e, end)
      }
    }
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
            bgcolor: 'var(--mortar-scrim)',
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
              gap: space.pad,
              textAlign: 'center',
              p: space.pad,
            })}
          >
            <Box sx={{ color: 'var(--mortar-accent-ink)', display: 'flex' }}>
              <Download size={80} strokeWidth={1.8} />
            </Box>
            {ready ? (
              <>
                <Typography sx={{ fontSize: 38, fontWeight: 700 }}>
                  {t`Drop to install into ${profile.name}`}
                </Typography>
                <Typography sx={{ maxWidth: 640, fontSize: 15, lineHeight: 1.5 }}>
                  {t`Zip, RAR, 7z, tar, gzip, xz, lzma, zstd and bzip2. Mortar reads the mods inside and shows you what was added.`}
                </Typography>
              </>
            ) : (
              <Typography sx={{ fontSize: 38, fontWeight: 700 }}>
                {locked ? t`Stop the game to change mods.` : t`Open a profile to install mods`}
              </Typography>
            )}
          </Box>
        </Box>
      ) : null}
    </>
  )
}
