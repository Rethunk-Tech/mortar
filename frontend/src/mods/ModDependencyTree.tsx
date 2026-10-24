import { useLingui } from '@lingui/react/macro'
import { Box, Button, Link, Typography } from '@mui/material'
import {
  Asterisk,
  CircleAlert,
  CircleCheck,
  CircleDashed,
  CircleSlash,
  type LucideIcon,
  TriangleAlert,
} from 'lucide-react'
import { memo, useDeferredValue, useMemo, useRef } from 'react'
import type { Broken } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { refWant } from '../queue/refWant.ts'
import { useQueue } from '../queue/store.ts'
import { pendingFor } from '../queue/totals.ts'
import { StableLabel } from '../shell/StableLabel.tsx'
import { sameValue } from '../shell/sameValue.ts'
import { Tooltip } from '../shell/Tooltip.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import {
  annotateTree,
  buildDependencyTrees,
  type DepViewNode,
  manifestsFromMods,
} from './dependencyTree.ts'
import { localId } from './dependents.ts'
import { useDetail } from './detail.ts'
import { problemsOf, sameId } from './lookup.ts'
import { openPage } from './menu.ts'
import { heading } from './paper.ts'
import { useRequirementName } from './requirementName.ts'
import { useMods } from './store.ts'

const noWrap = { whiteSpace: 'nowrap' } as const
const indentStep = 1.5
const rowMinHeight = 22
const rowGap = 0.75
const nameSize = 13
const headingGap = 0.5
const iconSize = 13

function MissingAdd({ id }: { id: string }) {
  const { t } = useLingui()
  const result = useMods((s) => s.problems)
  const queue = useQueue((s) => s.state.items)
  const profileId = useProfiles((s) => s.openId)
  const hit = problemsOf(result).find(
    (p) => p.kind === 'missing' && sameId(p.missing.id, id) && p.missing.reason !== 'disabled',
  )
  const where = hit?.kind === 'missing' ? hit.missing.where : undefined
  if (!where?.url) {
    return null
  }
  const { url } = where
  const pageButton = (
    <Button
      size="small"
      color="warning"
      variant="outlined"
      onClick={() => openPage(url)}
      sx={{ height: rowMinHeight, ...noWrap, flexShrink: 0 }}
    >
      {t`Open page`}
    </Button>
  )
  const want = refWant(where, 'dependency')
  if (!want) {
    return pageButton
  }
  const queued = pendingFor(queue, profileId, want.modId ?? 0, want.repo)
  return (
    <>
      {pageButton}
      <Button
        size="small"
        variant="contained"
        color="warning"
        disabled={queued}
        onClick={() => download([want]).catch(reportUnexpected)}
        sx={{ height: rowMinHeight, ...noWrap, flexShrink: 0 }}
      >
        <StableLabel labels={[t`Add`, t`Queued`]} shown={queued ? t`Queued` : t`Add`} />
      </Button>
    </>
  )
}

function stateTone(state: DepViewNode['state']): string {
  if (state === 'missing' || state === 'broken') {
    return 'warning.main'
  }
  if (state === 'disabled') {
    return 'text.secondary'
  }
  return 'success.main'
}

const stateIcon: Record<DepViewNode['state'], LucideIcon> = {
  enabled: CircleCheck,
  disabled: CircleSlash,
  missing: CircleAlert,
  broken: TriangleAlert,
}

function Mark({ icon: Icon, label, color }: { icon: LucideIcon; label: string; color: string }) {
  return (
    <Tooltip title={label}>
      <Box
        role="img"
        aria-label={label}
        tabIndex={0}
        sx={{ display: 'inline-flex', flexShrink: 0, mt: '3px', color }}
      >
        <Icon size={iconSize} aria-hidden={true} />
      </Box>
    </Tooltip>
  )
}

// Memoised, with Branch: a framework's Used by list runs to hundreds of rows, and the trees' nodes are kept until the
// mods or problems change, so a re-render of the panel around them skips every row.
const NodeRow = memo(
  function NodeRowView({ node, depth }: { node: DepViewNode; depth: number }) {
    const { t } = useLingui()
    const show = useDetail((s) => s.show)
    // Whether it is listed, not the mod itself: the store rebuilds every mod on each switch, and a node shows none of it.
    const listed = useMods((s) => s.mods.some((m) => sameId(m.id, node.id)))
    const label = useRequirementName(node.id)
    const mark = node.required ? t`Required` : t`Optional`
    const already = t`already listed`
    const edgeNote = node.cycle ? `${mark} · ${already}` : mark
    let status = t`Broken`
    if (node.state === 'enabled') {
      status = t`Enabled`
    } else if (node.state === 'disabled') {
      status = t`Disabled`
    } else if (node.state === 'missing') {
      status = t`Missing`
    }
    const tone = stateTone(node.state)
    return (
      <Box>
        <Box
          sx={{
            pl: depth * indentStep,
            minHeight: rowMinHeight,
            display: 'flex',
            alignItems: 'flex-start',
            columnGap: rowGap,
          }}
        >
          <Mark
            icon={node.required ? Asterisk : CircleDashed}
            label={edgeNote}
            color="text.secondary"
          />
          <Mark icon={stateIcon[node.state]} label={status} color={tone} />
          <Box
            sx={{
              minWidth: 0,
              display: 'flex',
              flexWrap: 'wrap',
              alignItems: 'center',
              columnGap: rowGap,
            }}
          >
            {listed ? (
              <Link
                component="button"
                onClick={() => {
                  const mod = useMods.getState().mods.find((m) => sameId(m.id, node.id))
                  if (mod) {
                    show(mod)
                  }
                }}
                sx={{
                  fontSize: nameSize,
                  color: 'text.secondary',
                  textDecoration: 'underline dotted',
                  textAlign: 'left',
                  overflowWrap: 'anywhere',
                }}
              >
                {label}
              </Link>
            ) : (
              <Typography
                title={localId(node.id)}
                sx={{ fontSize: nameSize, overflowWrap: 'anywhere' }}
              >
                {label}
              </Typography>
            )}
            {node.state === 'missing' ? <MissingAdd id={node.id} /> : null}
          </Box>
        </Box>
        {node.children.map((c) => (
          <NodeRow key={`${c.id}:${c.required}:${c.cycle}`} node={c} depth={depth + 1} />
        ))}
      </Box>
    )
  },
  (a, b) => a.depth === b.depth && sameNode(a.node, b.node),
)

// A switch rebuilds the trees, so a node is compared by what it shows rather than by identity.
function sameNode(a: DepViewNode, b: DepViewNode): boolean {
  return (
    a === b ||
    (a.id === b.id &&
      a.required === b.required &&
      a.cycle === b.cycle &&
      a.state === b.state &&
      a.children.length === b.children.length &&
      a.children.every((c, i) => {
        const d = b.children[i]
        return d !== undefined && sameNode(c, d)
      }))
  )
}

const Branch = memo(function BranchView({ title, nodes }: { title: string; nodes: DepViewNode[] }) {
  return (
    <Box>
      <Typography sx={{ fontSize: nameSize, color: 'text.secondary', mt: headingGap }}>
        {title}
      </Typography>
      {nodes.map((n) => (
        <NodeRow key={`${n.id}:${n.required}:${n.cycle}`} node={n} depth={0} />
      ))}
    </Box>
  )
})

function TreesView({
  mod,
  mods,
  broken,
}: {
  mod: Mod
  mods: Mod[]
  broken: Broken[] | null | undefined
}) {
  const { t } = useLingui()
  // The trees walk every mod in the profile, so they are rebuilt only when the mods or the problems change.
  const manifests = useMemo(() => manifestsFromMods(mods), [mods])
  const lastTrees = useRef<{ needs: DepViewNode[]; neededBy: DepViewNode[] } | null>(null)
  const built = useMemo(() => {
    const trees = buildDependencyTrees(manifests, mod.id)
    const installed = mods.map((m) => ({
      id: m.id,
      enabled: m.enabled,
      broken: (broken ?? []).some((b) => sameId(b.id, m.id)),
    }))
    return {
      needs: annotateTree(trees.needs, installed),
      neededBy: annotateTree(trees.neededBy, installed),
    }
  }, [manifests, mods, mod.id, broken])
  // A rebuild that changes nothing keeps the previous nodes, so the memoised rows skip it.
  const trees = lastTrees.current && sameValue(lastTrees.current, built) ? lastTrees.current : built
  lastTrees.current = trees
  const { needs, neededBy } = trees
  return (
    <Box>
      <Typography sx={heading}>{t`Dependencies`}</Typography>
      {needs.length === 0 && neededBy.length === 0 ? (
        <Typography sx={{ fontSize: nameSize, color: 'text.secondary' }}>{t`None`}</Typography>
      ) : null}
      {needs.length > 0 ? <Branch title={t`Needs`} nodes={needs} /> : null}
      {neededBy.length > 0 ? <Branch title={t`Used by`} nodes={neededBy} /> : null}
    </Box>
  )
}

const Trees = memo(TreesView)

// A toggle or a fresh check rebuilds the trees after the click has painted, not inside it.
export function ModDependencyTree({ mod }: { mod: Mod }) {
  const mods = useDeferredValue(useMods((s) => s.mods))
  const broken = useDeferredValue(useMods((s) => s.problems?.broken))
  return <Trees mod={mod} mods={mods} broken={broken} />
}
