import {
  closestCenter,
  DndContext,
  type DragEndEvent,
  DragOverlay,
  KeyboardSensor,
  PointerSensor,
  pointerWithin,
  type UniqueIdentifier,
  useSensor,
  useSensors,
} from '@dnd-kit/core'
import {
  SortableContext,
  type SortingStrategy,
  sortableKeyboardCoordinates,
  useSortable,
} from '@dnd-kit/sortable'
import { useLingui } from '@lingui/react/macro'
import { alpha, Box } from '@mui/material'
import { GripVertical } from 'lucide-react'
import { type ReactNode, useCallback, useMemo, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { SetModOrder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { useDragAnnouncements } from '../shell/dragAnnouncements.ts'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import type { ListRow } from './listColumns.ts'
import { modId } from './lookup.ts'
import { displayedKeys, droppedOrder } from './manualOrderLogic.ts'

const GHOST_TINT = 0.2
const HANDLE_PX = 14
const NONE = -1
const IDLE_OPACITY = 0.55

// Rows stay where they are while one is dragged: the virtualised list positions them itself.
const stay: SortingStrategy = () => null

function saveOrder(profile: Profile, keys: string[]) {
  const { game, replace } = useProfiles.getState()
  if (!game) {
    return
  }
  replace({ ...profile, modOrder: keys })
  SetModOrder(game.id, profile.id, keys)
    .then(replace)
    .catch((err: unknown) => {
      replace(profile)
      reportUnexpected(err)
    })
}

/** Lets the rows and cards below it be dragged into a new place while `enabled`. */
export function ManualOrder({
  enabled,
  profile,
  groups,
  children,
}: {
  enabled: boolean
  profile: Profile
  groups: readonly { items: readonly ListRow[] }[]
  children: ReactNode
}) {
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )
  const [lifted, setLifted] = useState<string | null>(null)
  const rows = useMemo(() => groups.flatMap((g) => g.items), [groups])
  const ids = useMemo(() => rows.map((r) => modId(r.mod)), [rows])
  const nameOf = useCallback(
    (id: UniqueIdentifier) => rows.find((r) => modId(r.mod) === id)?.mod.name ?? String(id),
    [rows],
  )
  const announcements = useDragAnnouncements(nameOf)
  if (!enabled) {
    return <>{children}</>
  }
  const rowOf = (id: unknown) => rows.find((r) => modId(r.mod) === id)
  const onDragEnd = ({ active, over }: DragEndEvent) => {
    setLifted(null)
    const from = rowOf(active.id)
    const to = over ? rowOf(over.id) : undefined
    if (!(from && to) || from.mod.key === to.mod.key) {
      return
    }
    saveOrder(
      profile,
      droppedOrder(displayedKeys(groups), profile.modOrder, from.mod.key, to.mod.key),
    )
  }
  return (
    <DndContext
      sensors={sensors}
      collisionDetection={(args) => {
        const within = pointerWithin(args)
        return within.length > 0 ? within : closestCenter(args)
      }}
      accessibility={{ announcements }}
      onDragStart={({ active }) => setLifted(rowOf(active.id)?.mod.name ?? null)}
      onDragCancel={() => setLifted(null)}
      onDragEnd={onDragEnd}
    >
      <SortableContext items={ids} strategy={stay}>
        {children}
      </SortableContext>
      <DragOverlay dropAnimation={null}>
        {lifted === null ? null : (
          <Box
            sx={(theme) => ({
              display: 'inline-flex',
              alignItems: 'center',
              height: 30,
              px: space.pad,
              borderRadius: '6px',
              border: `1px solid ${theme.palette.primary.main}`,
              bgcolor: alpha(theme.palette.primary.main, GHOST_TINT),
              fontWeight: 600,
              whiteSpace: 'nowrap',
              cursor: 'grabbing',
            })}
          >
            {lifted}
          </Box>
        )}
      </DragOverlay>
    </DndContext>
  )
}

/**
 * The drag handle of one row or card, laid over the start of its positioned parent, and the line where a drop would
 * land. It is the row's drop target too, so the row itself keeps its own click and keyboard handling.
 */
export function ManualHandle({ id, name }: { id: string; name: string }) {
  const { t } = useLingui()
  const { attributes, listeners, setNodeRef, isDragging, isOver, index, activeIndex } = useSortable(
    { id },
  )
  const after = activeIndex > NONE && activeIndex < index
  return (
    <Box
      sx={(theme) => ({
        position: 'absolute',
        inset: 0,
        pointerEvents: 'none',
        [after ? 'borderBottom' : 'borderTop']:
          isOver && !isDragging
            ? `2px solid ${theme.palette.primary.main}`
            : '2px solid transparent',
      })}
    >
      <Box
        component="button"
        type="button"
        ref={setNodeRef}
        {...attributes}
        {...listeners}
        aria-label={t`Move ${name}`}
        className="manual-handle"
        sx={{
          position: 'absolute',
          left: 0,
          top: 0,
          bottom: 0,
          width: HANDLE_PX,
          pointerEvents: 'auto',
          cursor: isDragging ? 'grabbing' : 'grab',
          opacity: isDragging ? 1 : IDLE_OPACITY,
          touchAction: 'none',
          // A plain button: every drag event redraws each mounted grip, and ButtonBase's ripple tree would go with it.
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          p: 0,
          border: 0,
          bgcolor: 'transparent',
          color: 'inherit',
          '&:focus-visible': { outline: '2px solid', outlineColor: 'primary.main' },
          '&:hover, &:focus-visible': { opacity: 1 },
        }}
      >
        <GripVertical size={HANDLE_PX} aria-hidden={true} />
      </Box>
    </Box>
  )
}
