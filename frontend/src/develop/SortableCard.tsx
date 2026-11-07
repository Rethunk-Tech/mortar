import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { useLingui } from '@lingui/react/macro'
import { alpha, Box, IconButton } from '@mui/material'
import { GripVertical } from 'lucide-react'
import type { DevProject } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { Tooltip } from '../shell/Tooltip.tsx'
import { ProjectCard } from './ProjectCard.tsx'

const DRAG_TINT_ALPHA = 0.24

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
      <ProjectCard
        game={game}
        project={project}
        list={list}
        handle={
          <Tooltip title={label}>
            <IconButton
              ref={setActivatorNodeRef}
              aria-label={label}
              {...attributes}
              {...listeners}
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
        }
      />
    </Box>
  )
}
