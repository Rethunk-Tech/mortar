import { useLingui } from '@lingui/react/macro'
import { Box, IconButton, Typography } from '@mui/material'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { useEffect, useState } from 'react'
import {
  FrameAssembly,
  GameCode,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import type { Member } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/dotnet/models.ts'
import type { StackFrame } from '../console/stackFrame.ts'
import { reportUnexpected } from '../toasts/report.ts'

const dotted = (type: string) => type.replaceAll('+', '.')

// The member a frame ran: in the assembly its module id names, else the game's own; the overload taking as many
// arguments as the frame lists when there are several.
async function memberOf(
  game: string,
  dir: string,
  f: StackFrame,
): Promise<{ assembly: string; member: Member } | null> {
  const assembly = f.mvid ? await FrameAssembly(game, dir, f.mvid) : ''
  const hits = (await GameCode(game, dir, assembly, `${f.type}::${f.method}`)) ?? []
  const named = hits.filter((m) => m.method === f.method && dotted(m.type) === dotted(f.type))
  const member = named.find((m) => (m.params ?? []).length === f.params) ?? named[0]
  return member ? { assembly, member } : null
}

/** Steps through a Console stack trace in Game code: each frame opens the method it ran. */
export function TraceSteps({
  game,
  dir,
  trace,
  onOpen,
}: {
  game: string
  dir: string
  trace: { frames: StackFrame[]; at: number }
  onOpen: (assembly: string, member: Member) => void
}) {
  const { t } = useLingui()
  const [at, setAt] = useState(trace.at)
  const [missing, setMissing] = useState(false)
  const frame = trace.frames[at]
  useEffect(() => {
    if (!frame) {
      return
    }
    let live = true
    memberOf(game, dir, frame)
      .then((found) => {
        if (!live) {
          return
        }
        setMissing(found === null)
        if (found) {
          onOpen(found.assembly, found.member)
        }
      })
      .catch(reportUnexpected)
    return () => {
      live = false
    }
  }, [game, dir, frame, onOpen])
  if (!frame) {
    return null
  }
  const name = `${dotted(frame.type).split('.').at(-1)}.${frame.method}`
  const n = at + 1
  const total = trace.frames.length
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', flexShrink: 0 }}>
      <IconButton
        size="small"
        aria-label={t`Previous frame`}
        disabled={at === 0}
        onClick={() => setAt(at - 1)}
      >
        <ChevronLeft size={16} />
      </IconButton>
      <Typography
        noWrap={true}
        title={`${dotted(frame.type)}.${frame.method}`}
        sx={{ fontSize: 'small.fontSize', color: missing ? 'text.secondary' : 'text.primary' }}
      >
        {missing
          ? t`Frame ${n} of ${total}: ${name} is not in the listed assemblies`
          : t`Frame ${n} of ${total}`}
      </Typography>
      <IconButton
        size="small"
        aria-label={t`Next frame`}
        disabled={at === trace.frames.length - 1}
        onClick={() => setAt(at + 1)}
      >
        <ChevronRight size={16} />
      </IconButton>
    </Box>
  )
}
