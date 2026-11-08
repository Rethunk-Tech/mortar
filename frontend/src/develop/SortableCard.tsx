import type { DraggableAttributes, DraggableSyntheticListeners } from '@dnd-kit/core'
import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { useLingui } from '@lingui/react/macro'
import { alpha, Box, IconButton } from '@mui/material'
import { GripVertical } from 'lucide-react'
import { createContext, useContext, useMemo } from 'react'
import type { DevProject } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { Tooltip } from '../shell/Tooltip.tsx'
import { ProjectCard } from './ProjectCard.tsx'

const DRAG_TINT_ALPHA = 0.24

type Grip = {
  label: string
  attributes: DraggableAttributes
  listeners: DraggableSyntheticListeners
  setActivatorNodeRef: (node: HTMLElement | null) => void
}

// Every drag event hands each sortable new attributes and listeners; reading them here keeps the card itself, a large
// tree, out of that redraw.
const GripContext = createContext<Grip | null>(null)

function ProjectGrip() {
  const grip = useContext(GripContext)
  if (!grip) {
    return null
  }
  return (
    <Tooltip title={grip.label}>
      <IconButton
        ref={grip.setActivatorNodeRef}
        aria-label={grip.label}
        {...grip.attributes}
        {...grip.listeners}
        sx={{
          width: 28,
          height: 40,
          borderRadius: '6px',
          color: 'text.secondary',
          cursor: 'grab',
          touchAction: 'none',
        }}
      >
        <GripVertical size={16} />
      </IconButton>
    </Tooltip>
  )
}

/** A project card that can be dragged to a new place by its grip, or moved with the keyboard once the grip is focused. */
export function SortableCard({
  game,
  project,
  list,
}: {
  game: string
  project: DevProject
  list: boolean
}) {
  const { t } = useLingui()
  const {
    attributes,
    listeners,
    setNodeRef,
    setActivatorNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: project.dir })
  const label = t`Reorder ${{ name: project.name }}`
  const grip = useMemo(
    () => ({ label, attributes, listeners, setActivatorNodeRef }),
    [label, attributes, listeners, setActivatorNodeRef],
  )
  const card = useMemo(
    () => <ProjectCard game={game} project={project} list={list} handle={<ProjectGrip />} />,
    [game, project, list],
  )
  return (
    <Box
      ref={setNodeRef}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      sx={{
        position: 'relative',
        zIndex: isDragging ? 1 : 0,
        borderRadius: '6px',
        outline: '2px solid',
        outlineColor: isDragging ? 'primary.main' : 'transparent',
        bgcolor: (th) => (isDragging ? alpha(th.palette.primary.main, DRAG_TINT_ALPHA) : undefined),
      }}
    >
      <GripContext value={grip}>{card}</GripContext>
    </Box>
  )
}
