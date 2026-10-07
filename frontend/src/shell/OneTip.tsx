import { Tooltip, type TooltipProps } from '@mui/material'
import { cloneElement, type MouseEvent, type ReactElement, useId } from 'react'
import { create } from 'zustand'

const useOpenTip = create<{ id: string | null; set: (id: string | null) => void }>((set) => ({
  id: null,
  set: (id) => set({ id }),
}))

// A Tooltip that closes the moment another OneTip opens, so moving across a row of icons (or menu rows) never
// leaves two showing while the first fades. A press closes it too.
export function OneTip({
  children,
  ...rest
}: Omit<TooltipProps, 'open' | 'onOpen' | 'onClose' | 'children'> & {
  children: ReactElement<{ onMouseDown?: (e: MouseEvent<HTMLElement>) => void }>
}) {
  const id = useId()
  const open = useOpenTip((s) => s.id === id)
  const { set } = useOpenTip.getState()
  return (
    <Tooltip
      {...rest}
      open={open}
      onOpen={() => set(id)}
      onClose={() => {
        if (useOpenTip.getState().id === id) {
          set(null)
        }
      }}
    >
      {cloneElement(children, {
        onMouseDown: (e: MouseEvent<HTMLElement>) => {
          children.props.onMouseDown?.(e)
          set(null)
        },
      })}
    </Tooltip>
  )
}
