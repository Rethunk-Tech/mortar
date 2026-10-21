import { useLingui } from '@lingui/react/macro'
import { Box, ButtonBase, IconButton, Typography } from '@mui/material'
import { PanelRightClose, PanelRightOpen } from 'lucide-react'
import { type RefObject, useEffect, useState } from 'react'
import { Tooltip } from '../../shell/Tooltip.tsx'
import { readStored, writeStored } from '../../shell/useStoredState.ts'
import { space } from '../../theme/density.ts'

const OPEN_KEY = 'mortar.configSectionNav'
const WIDTH_PX = 220
const ACTIVE_WEIGHT = 600
const IDLE_WEIGHT = 400
// A section counts as the one in view once its heading is within this far of the list's top.
const TOP_SLACK_PX = 24

const isBool = (v: unknown): v is boolean => typeof v === 'boolean'

const sectionEl = (list: HTMLElement, name: string) =>
  list.querySelector<HTMLElement>(`[data-section="${CSS.escape(name)}"]`)

// The section whose heading is at or above the top of the list, else the first.
function inView(list: HTMLElement, names: readonly string[]): string {
  const top = list.getBoundingClientRect().top + TOP_SLACK_PX
  let current = names[0] ?? ''
  for (const name of names) {
    const el = sectionEl(list, name)
    if (el && el.getBoundingClientRect().top <= top) {
      current = name
    }
  }
  return current
}

// The config file's sections down the right, for a long file: a click scrolls to one, and the section in view is
// marked. It follows the list's scroll itself, so the entries do not render again as it moves.
export function SectionNav({
  list,
  names,
}: {
  list: RefObject<HTMLElement | null>
  names: readonly string[]
}) {
  const { t } = useLingui()
  const [open, setOpen] = useState(() => readStored(OPEN_KEY, true, isBool))
  const [current, setCurrent] = useState(names[0] ?? '')
  useEffect(() => {
    const el = list.current
    if (!el) {
      return
    }
    const follow = () => setCurrent(inView(el, names))
    follow()
    el.addEventListener('scroll', follow, { passive: true })
    return () => el.removeEventListener('scroll', follow)
  }, [list, names])
  const toggle = () => {
    writeStored(OPEN_KEY, !open)
    setOpen(!open)
  }
  const label = open ? t`Hide sections` : t`Show sections`
  return (
    <Box
      component="nav"
      aria-label={t`Sections`}
      sx={{
        width: open ? WIDTH_PX : 'auto',
        flexShrink: 0,
        display: 'flex',
        flexDirection: 'column',
        minHeight: 0,
        borderLeft: '1px solid var(--mortar-hairline)',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, px: 0.5, py: 0.5 }}>
        <Tooltip title={label}>
          <IconButton size="small" aria-label={label} aria-expanded={open} onClick={toggle}>
            {open ? <PanelRightClose size={16} /> : <PanelRightOpen size={16} />}
          </IconButton>
        </Tooltip>
        {open ? (
          <Typography sx={{ fontSize: 12, fontWeight: 700, color: 'text.secondary' }}>
            {t`Sections`}
          </Typography>
        ) : null}
      </Box>
      {open ? (
        <Box sx={{ flex: 1, minHeight: 0, overflowY: 'auto', pb: space.gap }}>
          {names.map((name) => {
            const on = name === current
            return (
              <ButtonBase
                key={name}
                aria-current={on ? 'location' : undefined}
                onClick={() => sectionEl(list.current ?? document.body, name)?.scrollIntoView()}
                sx={{
                  display: 'block',
                  width: '100%',
                  textAlign: 'left',
                  px: 1.5,
                  py: 0.5,
                  fontSize: 13,
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                  borderLeft: '2px solid',
                  borderColor: on ? 'primary.main' : 'transparent',
                  color: on ? 'primary.main' : 'text.secondary',
                  fontWeight: on ? ACTIVE_WEIGHT : IDLE_WEIGHT,
                  '&:hover': { color: 'text.primary' },
                }}
                title={name}
              >
                {name}
              </ButtonBase>
            )
          })}
        </Box>
      ) : null}
    </Box>
  )
}
