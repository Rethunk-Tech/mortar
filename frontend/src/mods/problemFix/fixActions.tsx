import { useLingui } from '@lingui/react/macro'
import { Box, Button, Divider, ListItemText, ListSubheader, Menu, MenuItem } from '@mui/material'
import { ChevronDown } from 'lucide-react'
import {
  type MouseEvent,
  type ReactNode,
  useCallback,
  useContext,
  useId,
  useLayoutEffect,
  useRef,
  useState,
} from 'react'
import { Tooltip } from '../../shell/Tooltip.tsx'
import { LockedReason } from '../LockedReason.tsx'
import { arrangeFixes, type FixWeight } from './arrangeFixes.ts'
import { type Put, SplitContext } from './splitContext.ts'

interface FixChoice {
  label: string
  onClick: () => void
}

export interface FixSpec {
  label: string
  weight: FixWeight
  // A fix without onClick offers its choices: as the button it opens them, in the menu they list under its label.
  onClick?: () => void
  choices?: readonly FixChoice[]
  // The button's look: filled in the warning colour for a real problem, outlined for a cosmetic one.
  tone?: 'warning' | 'plain'
  // The running game holds the mod's files.
  locked?: boolean
  disabled?: boolean
  tooltip?: string
}

/** One fix of a problem row. Inside a SplitFix it joins that row's split button; elsewhere it is a button of its own. */
export function FixAction(spec: FixSpec) {
  const put = useContext(SplitContext)
  const id = useId()
  useLayoutEffect(() => put?.(id, spec))
  useLayoutEffect(() => () => put?.(id, null), [put, id])
  return put ? null : <LoneFix spec={spec} />
}

function LoneFix({ spec }: { spec: FixSpec }) {
  const variant = { main: 'contained', other: 'outlined', quiet: 'text' } as const
  const button = (
    <LockedReason locked={spec.locked === true}>
      <Button
        size="small"
        variant={variant[spec.weight]}
        color={spec.weight === 'quiet' || spec.tone === 'plain' ? 'inherit' : 'warning'}
        disabled={spec.locked === true || spec.disabled === true}
        onClick={spec.onClick}
        sx={{ flexShrink: 0 }}
      >
        {spec.label}
      </Button>
    </LockedReason>
  )
  return spec.tooltip ? <Tooltip title={spec.tooltip}>{button}</Tooltip> : button
}

// What the host draws depends on; a fix that re-renders with only new handlers leaves it alone.
const shapeOf = (specs: Map<string, FixSpec>) =>
  JSON.stringify(
    [...specs].map(([id, s]) => [
      id,
      s.label,
      s.weight,
      s.tone,
      s.locked,
      s.disabled,
      s.tooltip,
      s.choices?.map((c) => c.label),
    ]),
  )

const HALF_RADIUS = '4px'

/** A problem row's fixes as one split button: the heaviest fix, and an arrow opening the others, quiet ones last. The
 * fix components inside render their dialogs as usual and declare their buttons with FixAction. */
export function SplitFix({ children }: { children: ReactNode }) {
  const { t } = useLingui()
  const specs = useRef(new Map<string, FixSpec>())
  const [, setShape] = useState('')
  const [menu, setMenu] = useState<{ anchor: HTMLElement; only?: string } | null>(null)
  const put = useCallback<Put>((id, spec) => {
    if (spec) {
      specs.current.set(id, spec)
    } else {
      specs.current.delete(id)
    }
    setShape(shapeOf(specs.current))
  }, [])
  // Handlers are read at the click, so one a fix replaced without changing its shape is still the current one.
  const fire = (id: string, choice?: number) => {
    setMenu(null)
    const spec = specs.current.get(id)
    const run = choice === undefined ? spec?.onClick : spec?.choices?.[choice]?.onClick
    run?.()
  }
  const entries = [...specs.current].map(([id, spec]) => ({ id, ...spec }))
  const { main, menu: others, quiet } = arrangeFixes(entries)
  const item = (fix: (typeof entries)[number]) =>
    fix.choices
      ? [
          <ListSubheader key={`${fix.id}/head`}>{fix.label}</ListSubheader>,
          ...fix.choices.map((choice, i) => (
            <MenuItem
              key={`${fix.id}/${choice.label}`}
              disabled={fix.locked === true || fix.disabled === true}
              onClick={() => fire(fix.id, i)}
            >
              {choice.label}
            </MenuItem>
          )),
        ]
      : [
          <MenuItem
            key={fix.id}
            disabled={fix.locked === true || fix.disabled === true}
            onClick={() => fire(fix.id)}
          >
            <ListItemText primary={fix.label} secondary={fix.tooltip} />
          </MenuItem>,
        ]
  const menuItems = main
    ? [
        ...others.flatMap(item),
        ...(others.length > 0 && quiet.length > 0 ? [<Divider key="quiet" />] : []),
        ...quiet.flatMap(item),
      ]
    : []
  const shownChoices = menu?.only ? entries.find((fix) => fix.id === menu.only) : undefined
  // A fix whose choices the button opens shows just those, without its heading.
  const menuContent = shownChoices ? item(shownChoices).slice(1) : menuItems
  const variant = main?.tone === 'plain' ? 'outlined' : 'contained'
  const color = main?.tone === 'plain' ? 'inherit' : 'warning'
  return (
    <SplitContext.Provider value={put}>
      {children}
      {main ? (
        <Box sx={{ display: 'inline-flex', flexShrink: 0 }}>
          <LockedReason locked={main.locked === true}>
            <Tooltip title={main.tooltip ?? ''}>
              <Button
                size="small"
                variant={variant}
                color={color}
                disabled={main.locked === true || main.disabled === true}
                aria-haspopup={main.onClick ? undefined : 'menu'}
                onClick={(e: MouseEvent<HTMLButtonElement>) =>
                  main.onClick ? fire(main.id) : setMenu({ anchor: e.currentTarget, only: main.id })
                }
                sx={{
                  flexShrink: 0,
                  ...(menuItems.length > 0
                    ? { borderRadius: `${HALF_RADIUS} 0 0 ${HALF_RADIUS}` }
                    : {}),
                }}
              >
                {main.label}
              </Button>
            </Tooltip>
          </LockedReason>
          {menuItems.length > 0 ? (
            <Button
              size="small"
              variant={variant}
              color={color}
              aria-label={t`Other fixes`}
              aria-haspopup="menu"
              aria-expanded={menu !== null && !menu.only}
              onClick={(e) => setMenu({ anchor: e.currentTarget })}
              sx={{
                minWidth: 28,
                width: 28,
                px: 0,
                flexShrink: 0,
                borderRadius: `0 ${HALF_RADIUS} ${HALF_RADIUS} 0`,
                ...(variant === 'outlined'
                  ? { ml: '-1px' }
                  : { borderLeft: '1px solid var(--mortar-hairline-20)' }),
              }}
            >
              <ChevronDown size={15} aria-hidden={true} />
            </Button>
          ) : null}
        </Box>
      ) : (
        quiet.map((fix) => <LoneFix key={fix.id} spec={fix} />)
      )}
      <Menu anchorEl={menu?.anchor ?? null} open={menu !== null} onClose={() => setMenu(null)}>
        {menuContent}
      </Menu>
    </SplitContext.Provider>
  )
}
