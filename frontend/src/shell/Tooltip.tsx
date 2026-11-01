import { Tooltip as MuiTooltip, type TooltipProps } from '@mui/material'
import {
  cloneElement,
  type FocusEvent,
  type MouseEvent,
  type ReactElement,
  type SyntheticEvent,
  type TouchEvent,
  useEffect,
  useId,
  useRef,
  useState,
} from 'react'
import { create } from 'zustand'

// MUI's own delays.
const ENTER_MS = 100
const TOUCH_ENTER_MS = 700
const TOUCH_LEAVE_MS = 1500

// One tooltip shows at a time, so moving across a row of icons never leaves two showing while the first fades.
const useOpenTip = create<{ id: string | null }>(() => ({ id: null }))

interface ChildProps {
  onMouseEnter?: (e: MouseEvent<HTMLElement>) => void
  onMouseLeave?: (e: MouseEvent<HTMLElement>) => void
  onMouseDown?: (e: MouseEvent<HTMLElement>) => void
  onFocus?: (e: FocusEvent<HTMLElement>) => void
  onBlur?: (e: FocusEvent<HTMLElement>) => void
  onTouchStart?: (e: TouchEvent<HTMLElement>) => void
  onTouchEnd?: (e: TouchEvent<HTMLElement>) => void
}

/**
 * MUI's Tooltip, mounted beside its child only while it shows. MUI's keeps its child's node in state, so every
 * tooltip on a screen rendered itself and its child twice when the screen mounted; this one costs the child nothing
 * until it is hovered, focused from the keyboard or long-pressed. The child carries the accessible name or description
 * MUI would give it, its own props winning as in MUI, and a press closes the tooltip.
 */
export function Tooltip(props: TooltipProps) {
  const empty = props.title === '' || props.title === null || props.title === undefined
  if (empty) {
    return props.children
  }
  return props.open === undefined ? <LazyTooltip {...props} /> : <MuiTooltip {...props} />
}

function LazyTooltip({
  children,
  title,
  describeChild = false,
  disableHoverListener = false,
  onOpen,
  ...rest
}: TooltipProps) {
  const id = useId()
  const tipId = `${id}tip`
  const open = useOpenTip((s) => s.id === id)
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined)
  useEffect(
    () => () => {
      clearTimeout(timer.current)
      if (useOpenTip.getState().id === id) {
        useOpenTip.setState({ id: null })
      }
    },
    [id],
  )
  const show = (e: SyntheticEvent<HTMLElement>, ms: number) => {
    const el = e.currentTarget
    clearTimeout(timer.current)
    timer.current = setTimeout(() => {
      setAnchor(el)
      useOpenTip.setState({ id })
      onOpen?.(e)
    }, ms)
  }
  const hide = (ms = 0) => {
    clearTimeout(timer.current)
    const close = () => {
      if (useOpenTip.getState().id === id) {
        useOpenTip.setState({ id: null })
      }
    }
    if (ms > 0) {
      timer.current = setTimeout(close, ms)
    } else {
      close()
    }
  }
  const child = children as ReactElement<ChildProps>
  const own = child.props
  const titleIsString = typeof title === 'string'
  const named = describeChild
    ? {
        // Not title: the browser's own tooltip would show beside this one.
        'aria-description': !open && titleIsString ? title : undefined,
        'aria-describedby': open ? tipId : undefined,
      }
    : {
        'aria-label': titleIsString ? title : undefined,
        'aria-labelledby': open && !titleIsString ? tipId : undefined,
      }
  return (
    <>
      {cloneElement(child, {
        ...named,
        ...own,
        onMouseEnter: (e: MouseEvent<HTMLElement>) => {
          own.onMouseEnter?.(e)
          if (!disableHoverListener) {
            show(e, ENTER_MS)
          }
        },
        onMouseLeave: (e: MouseEvent<HTMLElement>) => {
          own.onMouseLeave?.(e)
          hide()
        },
        onMouseDown: (e: MouseEvent<HTMLElement>) => {
          own.onMouseDown?.(e)
          hide()
        },
        onFocus: (e: FocusEvent<HTMLElement>) => {
          own.onFocus?.(e)
          if (e.currentTarget.matches(':focus-visible')) {
            show(e, 0)
          }
        },
        onBlur: (e: FocusEvent<HTMLElement>) => {
          own.onBlur?.(e)
          hide()
        },
        onTouchStart: (e: TouchEvent<HTMLElement>) => {
          own.onTouchStart?.(e)
          show(e, TOUCH_ENTER_MS)
        },
        onTouchEnd: (e: TouchEvent<HTMLElement>) => {
          own.onTouchEnd?.(e)
          hide(TOUCH_LEAVE_MS)
        },
      })}
      {open && anchor ? (
        <MuiTooltip
          {...rest}
          id={tipId}
          title={title}
          open={true}
          describeChild={describeChild}
          slotProps={{ popper: { anchorEl: anchor } }}
        >
          <span hidden={true} />
        </MuiTooltip>
      ) : null}
    </>
  )
}
