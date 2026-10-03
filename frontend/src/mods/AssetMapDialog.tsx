import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  LinearProgress,
  TextField,
  Typography,
} from '@mui/material'
import { useEffect, useState } from 'react'
import type { AssetTarget } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/models.ts'
import { AssetMap } from '../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { paper } from './paper.ts'

const filterDelayMs = 200

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
              <Typography
                title={
                  many
                    ? t`More than one mod changes this; the winner's version is used.`
                    : undefined
                }
                sx={{
                  fontSize: 13,
                  color: many ? 'warning.main' : 'text.secondary',
                  whiteSpace: 'nowrap',
                }}
              >
                {plural(count, { one: 'Changed by # mod', other: 'Changed by # mods' })}
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

export function AssetMapDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game)
  const openId = useProfiles((s) => s.openId)
  const [filter, setFilter] = useState('')
  const [mapTargets, setMapTargets] = useState<AssetTarget[] | null>(null)
  const [expanded, setExpanded] = useState<string | null>(null)

  useEffect(() => {
    if (!(open && game && openId)) {
      return
    }
    const handle = globalThis.setTimeout(() => {
      AssetMap(game.id, openId, filter.trim(), 0)
        .then((page) => {
          setMapTargets(page.targets ?? [])
        })
        .catch(reportUnexpected)
    }, filterDelayMs)
    return () => {
      globalThis.clearTimeout(handle)
    }
  }, [open, game, openId, filter])

  return (
    <Dialog
      open={open}
      onClose={onClose}
      maxWidth="md"
      fullWidth={true}
      scroll="paper"
      transitionDuration={0}
      slotProps={{ paper }}
    >
      <DialogTitle>{t`Asset map`}</DialogTitle>
      <DialogContent dividers={true} sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
        <Typography sx={{ fontSize: 14, color: 'text.secondary' }}>
          {t`Every game asset the profile's content packs change, and which mods change it. Open one to see each mod's edit.`}
        </Typography>
        <TextField
          size="small"
          value={filter}
          onChange={(e) => {
            setFilter(e.target.value)
          }}
          placeholder={t`Find an asset, for example Maps/Town`}
          slotProps={{ htmlInput: { 'aria-label': t`Find an asset` } }}
        />
        {mapTargets === null ? (
          <LinearProgress />
        ) : (
          <TargetList
            targets={mapTargets}
            open={expanded}
            onToggle={(id) => {
              setExpanded((cur) => (cur === id ? null : id))
            }}
          />
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Close`}</Button>
      </DialogActions>
    </Dialog>
  )
}
