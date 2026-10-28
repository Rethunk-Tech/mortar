import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  ButtonBase,
  Chip,
  FormControlLabel,
  LinearProgress,
  Switch,
  Typography,
} from '@mui/material'
import { Map as MapIcon } from 'lucide-react'
import { useState } from 'react'
import type {
  AssetTarget,
  AssetTouch,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/framework/contentpatcher/models.ts'
import { AppDialog, DialogFooter, ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useAssetPages } from './assetPages.ts'
import { LockedReason } from './LockedReason.tsx'
import { sameId } from './lookup.ts'
import { LinkedText } from './ModLinks.tsx'
import { applyWins } from './problemFix/applyWins.ts'
import { useMods } from './store.ts'
import { useLocked } from './useLocked.ts'

const chipSx = { height: 22, borderRadius: '4px' } as const
const buttonSx = { height: 24, flexShrink: 0, whiteSpace: 'nowrap' } as const

function TouchActions({
  target,
  mod,
  onChanged,
}: {
  target: AssetTarget
  mod: AssetTouch
  onChanged: () => void
}) {
  const { t } = useLingui()
  const locked = useLocked()
  const mods = useMods((s) => s.mods)
  const setEnabled = useMods((s) => s.setEnabled)
  if (mod.winner) {
    return null
  }
  const entry = mods.find((m) => m.key === mod.modKey && sameId(m.id, mod.modId))
  const others = (target.mods ?? [])
    .map((m) => m.modId)
    .filter((id, i, ids) => !sameId(id, mod.modId) && ids.findIndex((x) => sameId(x, id)) === i)
  return (
    <LockedReason locked={locked}>
      <Box sx={{ display: 'flex', gap: 0.5, ml: 'auto' }}>
        {mod.canWin ? (
          <Button
            size="small"
            variant="outlined"
            color="inherit"
            disabled={locked}
            title={t`Load ${{ name: mod.modName }} after the other mods here, so its edit applies last`}
            onClick={() => {
              applyWins(mod.modKey, mod.modId, others, true).then(onChanged).catch(reportUnexpected)
            }}
            sx={buttonSx}
          >
            {t`Make win`}
          </Button>
        ) : null}
        {entry ? (
          <Button
            size="small"
            color="inherit"
            variant="text"
            disabled={locked}
            onClick={() => {
              setEnabled(entry, false)
                .then((off) => (off.length > 0 ? onChanged() : undefined))
                .catch(reportUnexpected)
            }}
            sx={buttonSx}
          >
            {t`Disable`}
          </Button>
        ) : null}
      </Box>
    </LockedReason>
  )
}

function ModTouches({ target, onChanged }: { target: AssetTarget; onChanged: () => void }) {
  const { t } = useLingui()
  const many = new Set((target.mods ?? []).map((m) => m.modId.toLowerCase())).size > 1
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, mt: 0.75 }}>
      {(target.mods ?? []).map((mod) => {
        const name = mod.modName || mod.modId
        return (
          <Box
            key={`${mod.modId}-${mod.action}-${String(mod.index)}-${mod.source ?? ''}-${mod.dataKey ?? ''}`}
            sx={{ display: 'flex', alignItems: 'center', gap: 0.75, flexWrap: 'wrap' }}
          >
            <Typography
              sx={{ fontSize: 'base.fontSize', fontWeight: mod.winner ? 'bold' : 'normal' }}
            >
              <LinkedText text={name} links={[{ name, key: mod.modKey, id: mod.modId }]} />
            </Typography>
            {mod.action ? <Chip size="small" label={mod.action} sx={chipSx} /> : null}
            {mod.priority ? (
              <Chip size="small" variant="outlined" label={mod.priority} sx={chipSx} />
            ) : null}
            {mod.winner ? (
              <Chip size="small" color="success" label={t`winner`} sx={chipSx} />
            ) : null}
            {many ? <TouchActions target={target} mod={mod} onChanged={onChanged} /> : null}
          </Box>
        )
      })}
    </Box>
  )
}

function TargetRow({
  target,
  expanded,
  onToggle,
  onChanged,
}: {
  target: AssetTarget
  expanded: boolean
  onToggle: () => void
  onChanged: () => void
}) {
  const { t } = useLingui()
  const count = new Set((target.mods ?? []).map((m) => m.modId.toLowerCase())).size
  const many = count > 1
  const label = target.key ? `${target.target} ${target.key}` : target.target
  return (
    <Box sx={{ bgcolor: 'background.paper', borderRadius: 1, px: space.gap, py: space.gap }}>
      <ButtonBase
        aria-expanded={expanded}
        onClick={onToggle}
        sx={{
          display: 'flex',
          alignItems: 'baseline',
          justifyContent: 'flex-start',
          gap: space.gap,
          width: '100%',
          textAlign: 'left',
        }}
      >
        <Typography component="span" sx={{ fontSize: 'medium.fontSize', fontWeight: 600, flex: 1 }}>
          {label}
        </Typography>
        <Typography
          component="span"
          title={
            many ? t`More than one mod changes this; the winner's version is used.` : undefined
          }
          sx={{
            fontSize: 'base.fontSize',
            color: many ? 'warning.main' : 'text.secondary',
            whiteSpace: 'nowrap',
          }}
        >
          {plural(count, { one: 'Changed by # mod', other: 'Changed by # mods' })}
        </Typography>
      </ButtonBase>
      {expanded ? <ModTouches target={target} onChanged={onChanged} /> : null}
    </Box>
  )
}

// focus opens the map on one asset: filtered to it with its rows expanded, as a conflict's Show conflicts does.
export function AssetMapDialog({
  open,
  onClose,
  focus,
}: {
  open: boolean
  onClose: () => void
  focus?: string
}) {
  const { t } = useLingui()
  const [filter, setFilter] = useState(focus ?? '')
  const [shared, setShared] = useState(true)
  // undefined until the user opens or closes a row, so the focused asset starts open.
  const [expanded, setExpanded] = useState<string | null | undefined>(undefined)
  const { page, more, reload } = useAssetPages(open, filter, shared)
  const targets = page?.targets ?? []
  return (
    <AppDialog
      open={open}
      onClose={onClose}
      size="lg"
      title={t`Asset map`}
      subtitle={t`Every game asset the profile's mods change, and which mod wins it. Open one to see each mod's edit in load order.`}
      icon={
        <ToneIcon tone="neutral">
          <MapIcon size={22} />
        </ToneIcon>
      }
      footer={
        <DialogFooter>
          <Button onClick={onClose}>{t`Close`}</Button>
        </DialogFooter>
      }
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
        <SearchField
          label={t`Find an asset`}
          placeholder={t`Find an asset, for example Maps/Town`}
          value={filter}
          onChange={setFilter}
        />
        <Box sx={{ display: 'flex', alignItems: 'center', gap: space.pad, flexWrap: 'wrap' }}>
          <FormControlLabel
            control={<Switch size="small" checked={shared} onChange={(_, on) => setShared(on)} />}
            label={t`Only assets more than one mod changes`}
          />
          {page ? (
            <Typography sx={{ fontSize: 'base.fontSize', color: 'text.secondary', ml: 'auto' }}>
              {t`${plural({ count: page.all }, { one: '# asset', other: '# assets' })}, ${{ shared: page.shared }} changed by more than one mod`}
            </Typography>
          ) : null}
        </Box>
        {page === null ? (
          <LinearProgress />
        ) : (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.75 }}>
            {targets.map((target) => {
              const id = `${target.target}\t${target.key ?? ''}`
              const isOpen =
                expanded === undefined
                  ? focus !== undefined && target.target.toLowerCase() === focus.toLowerCase()
                  : expanded === id
              return (
                <TargetRow
                  key={id}
                  target={target}
                  expanded={isOpen}
                  onToggle={() => setExpanded(isOpen ? null : id)}
                  onChanged={reload}
                />
              )
            })}
            {targets.length < page.total ? (
              <Button color="inherit" onClick={more} sx={{ alignSelf: 'center' }}>
                {t`Show more`}
              </Button>
            ) : null}
          </Box>
        )}
      </Box>
    </AppDialog>
  )
}
