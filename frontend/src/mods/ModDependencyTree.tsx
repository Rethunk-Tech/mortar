import { useLingui } from '@lingui/react/macro'
import { Box, Button, Link, Typography } from '@mui/material'
import { memo, useDeferredValue, useMemo } from 'react'
import type { Broken } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/problems/models.ts'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { refWant } from '../queue/refWant.ts'
import { useQueue } from '../queue/store.ts'
import { pendingFor } from '../queue/totals.ts'
import { StableLabel } from '../shell/StableLabel.tsx'
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
const captionSize = 11
const nameSize = 13
const headingGap = 0.5
const caption = { fontSize: captionSize, flexShrink: 0 } as const

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

function stateTone(state: DepViewNode['state']): string | undefined {
  if (state === 'missing' || state === 'broken') {
    return 'warning.main'
  }
  if (state === 'disabled') {
    return 'text.secondary'
  }
  return undefined
}

function NodeRow({ node, mods, depth }: { node: DepViewNode; mods: Mod[]; depth: number }) {
  const { t } = useLingui()
  const show = useDetail((s) => s.show)
  const listed = mods.find((m) => sameId(m.id, node.id))
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
      <Box sx={{ pl: depth * indentStep, minHeight: rowMinHeight }}>
        {listed ? (
          <Link
            component="button"
            onClick={() => show(listed)}
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
        <Box
          sx={{
            display: 'flex',
            flexWrap: 'wrap',
            alignItems: 'center',
            columnGap: rowGap,
            rowGap: 0.5,
          }}
        >
          <Typography sx={{ ...caption, color: 'text.secondary' }}>{edgeNote}</Typography>
          <Typography sx={{ ...caption, color: tone }}>{status}</Typography>
          {node.state === 'missing' ? <MissingAdd id={node.id} /> : null}
        </Box>
      </Box>
      {node.children.map((c) => (
        <NodeRow key={`${c.id}:${c.required}:${c.cycle}`} node={c} mods={mods} depth={depth + 1} />
      ))}
    </Box>
  )
}

function Branch({ title, nodes, mods }: { title: string; nodes: DepViewNode[]; mods: Mod[] }) {
  const { t } = useLingui()
  return (
    <Box>
      <Typography sx={{ ...heading, mt: headingGap }}>{title}</Typography>
      {nodes.length === 0 ? (
        <Typography sx={{ fontSize: nameSize, color: 'text.secondary' }}>{t`None`}</Typography>
      ) : (
        nodes.map((n) => (
          <NodeRow key={`${n.id}:${n.required}:${n.cycle}`} node={n} mods={mods} depth={0} />
        ))
      )}
    </Box>
  )
}

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
  const { needs, neededBy } = useMemo(() => {
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
  return (
    <Box>
      <Typography sx={heading}>{t`Dependencies`}</Typography>
      <Branch title={t`Needs`} nodes={needs} mods={mods} />
      <Branch title={t`Used by`} nodes={neededBy} mods={mods} />
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
