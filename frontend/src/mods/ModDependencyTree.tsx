import { useLingui } from '@lingui/react/macro'
import { Box, Button, Link, Typography } from '@mui/material'
import {
  Asterisk,
  ChevronDown,
  ChevronRight,
  CircleAlert,
  CircleCheck,
  CircleDashed,
  CircleSlash,
  type LucideIcon,
  TriangleAlert,
} from 'lucide-react'
import { memo, useCallback, useDeferredValue, useMemo, useState } from 'react'
import type { Broken } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { refWant } from '../queue/refWant.ts'
import { useQueue } from '../queue/store.ts'
import { pendingFor } from '../queue/totals.ts'
import { StableLabel } from '../shell/StableLabel.tsx'
import { Tooltip } from '../shell/Tooltip.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import {
  annotateTree,
  buildDependencyTrees,
  childrenOf,
  type DepViewNode,
  manifestsFromMods,
} from './dependencyTree.ts'
import { localId } from './dependents.ts'
import { useDetail } from './detail.ts'
import { modById, problemsOf, sameId } from './lookup.ts'
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
// Rows rendered before Show all: a framework's direct dependents run to hundreds.
const firstRows = 50

type Expand = (node: DepViewNode) => DepViewNode[]

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

function NodeList({
  nodes,
  depth,
  expand,
}: {
  nodes: DepViewNode[]
  depth: number
  expand: Expand
}) {
  const { t } = useLingui()
  const [all, setAll] = useState(false)
  const total = nodes.length
  const shown = all ? nodes : nodes.slice(0, firstRows)
  return (
    <>
      {shown.map((n) => (
        <NodeRow key={`${n.id}:${n.required}`} node={n} depth={depth} expand={expand} />
      ))}
      {shown.length < nodes.length ? (
        <Button
          size="small"
          onClick={() => setAll(true)}
          sx={{ ml: depth * indentStep, ...noWrap }}
        >
          {t`Show all ${total}`}
        </Button>
      ) : null}
    </>
  )
}

// Memoised: a node's children are built only when it is opened, and the memo keeps an unrelated re-render of the panel
// from touching rows.
const NodeRow = memo(
  function NodeRowView({
    node,
    depth,
    expand,
  }: {
    node: DepViewNode
    depth: number
    expand: Expand
  }) {
    const { t } = useLingui()
    const [open, setOpen] = useState(false)
    const children = useMemo(() => (open ? expand(node) : []), [open, expand, node])
    const show = useDetail((s) => s.show)
    // Whether it is listed, not the mod itself: the store rebuilds every mod on each switch, and a node shows none of it.
    const listed = useMods((s) => modById(s.mods, node.id) !== undefined)
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
          {node.count > 0 ? (
            <Link
              component="button"
              aria-expanded={open}
              aria-label={open ? t`Collapse` : t`Expand`}
              onClick={() => setOpen((v) => !v)}
              sx={{ display: 'inline-flex', mt: '3px', color: 'text.secondary' }}
            >
              {open ? (
                <ChevronDown size={iconSize} aria-hidden={true} />
              ) : (
                <ChevronRight size={iconSize} aria-hidden={true} />
              )}
            </Link>
          ) : (
            <Box sx={{ width: iconSize, flexShrink: 0 }} />
          )}
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
                  const mod = modById(useMods.getState().mods, node.id)
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
            {node.count > 0 ? (
              <Typography sx={{ fontSize: nameSize, color: 'text.secondary' }}>
                {node.count}
              </Typography>
            ) : null}
            {node.state === 'missing' ? <MissingAdd id={node.id} /> : null}
          </Box>
        </Box>
        {open ? <NodeList nodes={children} depth={depth + 1} expand={expand} /> : null}
      </Box>
    )
  },
  (a, b) =>
    a.depth === b.depth &&
    a.expand === b.expand &&
    a.node.id === b.node.id &&
    a.node.required === b.node.required &&
    a.node.cycle === b.node.cycle &&
    a.node.count === b.node.count &&
    a.node.state === b.node.state,
)

const Branch = memo(function BranchView({
  title,
  nodes,
  expand,
}: {
  title: string
  nodes: DepViewNode[]
  expand: Expand
}) {
  return (
    <Box>
      <Typography sx={{ fontSize: nameSize, color: 'text.secondary', mt: headingGap }}>
        {title}
      </Typography>
      <NodeList nodes={nodes} depth={0} expand={expand} />
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
  const installed = useMemo(
    () =>
      mods.map((m) => ({
        id: m.id,
        enabled: m.enabled,
        broken: (broken ?? []).some((b) => sameId(b.id, m.id)),
      })),
    [mods, broken],
  )
  const { needs, neededBy } = useMemo(() => {
    const trees = buildDependencyTrees(manifests, mod.id)
    return {
      needs: annotateTree(trees.needs, installed),
      neededBy: annotateTree(trees.neededBy, installed),
    }
  }, [manifests, installed, mod.id])
  const expandNeeds = useCallback<Expand>(
    (n) => annotateTree(childrenOf(manifests, 'needs', n), installed),
    [manifests, installed],
  )
  const expandUsedBy = useCallback<Expand>(
    (n) => annotateTree(childrenOf(manifests, 'neededBy', n), installed),
    [manifests, installed],
  )
  return (
    <Box>
      <Typography sx={heading}>{t`Dependencies`}</Typography>
      {needs.length === 0 && neededBy.length === 0 ? (
        <Typography sx={{ fontSize: nameSize, color: 'text.secondary' }}>{t`None`}</Typography>
      ) : null}
      {needs.length > 0 ? <Branch title={t`Needs`} nodes={needs} expand={expandNeeds} /> : null}
      {neededBy.length > 0 ? (
        <Branch title={t`Used by`} nodes={neededBy} expand={expandUsedBy} />
      ) : null}
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
