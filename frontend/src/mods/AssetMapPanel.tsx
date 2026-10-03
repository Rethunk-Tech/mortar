import { useLingui } from '@lingui/react/macro'
import { Box, Chip, TextField, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import type { AssetTarget } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import {
  AssetMap,
  WhoChanges,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'

const whoQueryDelayMs = 200

function TargetList({
  targets,
  open,
  onToggle,
}: {
  targets: AssetTarget[]
  open?: string | null
  onToggle?: (id: string) => void
}) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.75 }}>
      {targets.map((target) => {
        const id = `${target.target}\t${target.key ?? ''}`
        const count = (target.mods ?? []).length
        const many = count > 1
        const expanded = !onToggle || open === id
        const label = target.key ? `${target.target} ${target.key}` : target.target
        return (
          <Box
            key={id}
            sx={{
              bgcolor: 'background.paper',
              borderRadius: 1,
              px: 1.25,
              py: 1,
              outline: many ? '1px solid' : 'none',
              outlineColor: 'warning.main',
            }}
          >
            <Box
              component={onToggle ? 'button' : 'div'}
              type={onToggle ? 'button' : undefined}
              onClick={
                onToggle
                  ? () => {
                      onToggle(id)
                    }
                  : undefined
              }
              sx={{
                display: 'flex',
                alignItems: 'baseline',
                gap: 1,
                width: '100%',
                bgcolor: 'transparent',
                border: 0,
                color: 'inherit',
                textAlign: 'left',
                p: 0,
                cursor: onToggle ? 'pointer' : 'default',
              }}
            >
              <Typography sx={{ fontSize: 14, fontWeight: 600, flex: 1 }}>{label}</Typography>
              <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
                {t`${count} mods`}
              </Typography>
            </Box>
            {expanded ? <ModTouches target={target} /> : null}
          </Box>
        )
      })}
    </Box>
  )
}

function ModTouches({ target }: { target: AssetTarget }) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, mt: 0.75 }}>
      {(target.mods ?? []).map((mod) => (
        <Box
          key={`${mod.modId}-${mod.action}-${String(mod.index)}-${mod.source ?? ''}-${mod.dataKey ?? ''}`}
          sx={{ display: 'flex', alignItems: 'center', gap: 0.75, flexWrap: 'wrap' }}
        >
          <Typography sx={{ fontSize: 13, fontWeight: mod.winner ? 'bold' : 'normal' }}>
            {mod.modName || mod.modId}
          </Typography>
          {mod.action ? (
            <Chip size="small" label={mod.action} sx={{ height: 22, borderRadius: '4px' }} />
          ) : null}
          {mod.winner ? (
            <Chip
              size="small"
              color="success"
              label={t`winner`}
              sx={{ height: 22, borderRadius: '4px' }}
            />
          ) : null}
        </Box>
      ))}
    </Box>
  )
}

export function AssetMapPanel() {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game)
  const openId = useProfiles((s) => s.openId)
  const [query, setQuery] = useState('')
  const [filter, setFilter] = useState('')
  const [who, setWho] = useState<AssetTarget[]>([])
  const [mapTargets, setMapTargets] = useState<AssetTarget[]>([])
  const [open, setOpen] = useState<string | null>(null)

  useEffect(() => {
    if (!(game && openId)) {
      return
    }
    const q = query.trim()
    if (q === '') {
      setWho([])
      return
    }
    const handle = globalThis.setTimeout(() => {
      WhoChanges(game.id, openId, q)
        .then((page) => {
          setWho(page.targets ?? [])
        })
        .catch(reportUnexpected)
    }, whoQueryDelayMs)
    return () => {
      globalThis.clearTimeout(handle)
    }
  }, [game, openId, query])

  useEffect(() => {
    if (!(game && openId)) {
      return
    }
    AssetMap(game.id, openId, filter.trim(), 0)
      .then((page) => {
        setMapTargets(page.targets ?? [])
      })
      .catch(reportUnexpected)
  }, [game, openId, filter])

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, mb: 2 }}>
      <TextField
        size="small"
        value={query}
        onChange={(e) => {
          setQuery(e.target.value)
        }}
        placeholder={t`Who changes this?`}
        slotProps={{ htmlInput: { 'aria-label': t`Who changes this?` } }}
      />
      {who.length > 0 ? <TargetList targets={who} /> : null}
      <TextField
        size="small"
        value={filter}
        onChange={(e) => {
          setFilter(e.target.value)
        }}
        placeholder={t`Filter assets`}
        slotProps={{ htmlInput: { 'aria-label': t`Filter assets` } }}
      />
      <TargetList
        targets={mapTargets}
        open={open}
        onToggle={(id) => {
          setOpen((cur) => (cur === id ? null : id))
        }}
      />
    </Box>
  )
}
