import { useLingui } from '@lingui/react/macro'
import { Box, Button, Link, Typography } from '@mui/material'
import type { Mod } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { refWant } from '../queue/refWant.ts'
import { useQueue } from '../queue/store.ts'
import { pendingFor } from '../queue/totals.ts'
import { reportUnexpected } from '../toasts/report.ts'
import {
  annotateTree,
  buildDependencyTrees,
  type DepViewNode,
  manifestsFromMods,
} from './dependencyTree.ts'
import { useDetail } from './detail.ts'
import { problemsOf, sameId } from './lookup.ts'
import { openPage } from './menu.ts'
import { heading } from './paper.ts'
import { useMods } from './store.ts'

const noWrap = { whiteSpace: 'nowrap' } as const
const indentStep = 1.5
const rowMinHeight = 22
const rowGap = 0.75
const captionSize = 11
const nameSize = 13
const headingGap = 0.5
const caption = { fontSize: captionSize, flexShrink: 0 } as const

function MissingAdd({ uniqueID }: { uniqueID: string }) {
  const { t } = useLingui()
  const result = useMods((s) => s.problems)
  const queue = useQueue((s) => s.state.items)
  const profileId = useProfiles((s) => s.openId)
  const hit = problemsOf(result).find(
    (p) =>
      p.kind === 'missing' &&
      sameId(p.missing.uniqueId, uniqueID) &&
      p.missing.reason !== 'disabled',
  )
  const where = hit?.kind === 'missing' ? hit.missing.where : undefined
  if (!where?.url) {
    return null
  }
  const { url } = where
  const github = where.site === 'GitHub' && where.github !== ''
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
        {queued ? t`Queued` : t`Add`}
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
  const listed = mods.find((m) => sameId(m.uniqueId, node.uniqueID))
  const label = listed?.name || node.uniqueID
  const mark = node.required ? t`Required` : t`Optional`
  const already = t`already listed`
  const edgeNote = node.cycle ? `${mark} · ${already}` : mark
  let status = t`Broken`
  if (node.state === 'enabled') {
    status = t`Enabled`
  } else if (node.state === 'disabled') {
    status = t`Off`
  } else if (node.state === 'missing') {
    status = t`Missing`
  }
  const tone = stateTone(node.state)
  return (
    <Box>
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: rowGap,
          pl: depth * indentStep,
          minHeight: rowMinHeight,
        }}
      >
        {listed ? (
          <Link
            component="button"
            onClick={() => show(listed)}
            sx={{
              fontSize: nameSize,
              color: 'text.secondary',
              textDecoration: 'underline dotted',
              textAlign: 'left',
              minWidth: 0,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {label}
          </Link>
        ) : (
          <Typography noWrap={true} sx={{ fontSize: nameSize, minWidth: 0 }}>
            {label}
          </Typography>
        )}
        <Typography noWrap={true} sx={{ ...caption, color: 'text.secondary' }}>
          {edgeNote}
        </Typography>
        <Typography noWrap={true} sx={{ ...caption, color: tone }}>
          {status}
        </Typography>
        {node.state === 'missing' ? <MissingAdd uniqueID={node.uniqueID} /> : null}
      </Box>
      {node.children.map((c) => (
        <NodeRow
          key={`${c.uniqueID}:${c.required}:${c.cycle}`}
          node={c}
          mods={mods}
          depth={depth + 1}
        />
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
          <NodeRow key={`${n.uniqueID}:${n.required}:${n.cycle}`} node={n} mods={mods} depth={0} />
        ))
      )}
    </Box>
  )
}

export function ModDependencyTree({ mod }: { mod: Mod }) {
  const { t } = useLingui()
  const mods = useMods((s) => s.mods)
  const broken = useMods((s) => s.problems?.broken)
  const trees = buildDependencyTrees(manifestsFromMods(mods), mod.uniqueId)
  const installed = mods.map((m) => ({
    uniqueID: m.uniqueId,
    enabled: m.enabled,
    broken: (broken ?? []).some((b) => sameId(b.uniqueId, m.uniqueId)),
  }))
  const needs = annotateTree(trees.needs, installed)
  const neededBy = annotateTree(trees.neededBy, installed)
  return (
    <Box>
      <Typography sx={heading}>{t`Dependencies`}</Typography>
      <Branch title={t`Needs`} nodes={needs} mods={mods} />
      <Branch title={t`Needed by`} nodes={neededBy} mods={mods} />
    </Box>
  )
}
