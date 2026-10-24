import { useSortable } from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { useLingui } from '@lingui/react/macro'
import { alpha, Box, Button, IconButton, Menu, Typography } from '@mui/material'
import { GripVertical, MoreHorizontal, Palette, Pencil, Share2 } from 'lucide-react'
import { memo, useRef, useState } from 'react'
import type { ProfileBudget } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/datasvc/models.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { NameField } from '../game/NameField.tsx'
import { CoverMenuItems, MoreMenuItems, ProfileMenuItem } from '../game/ProfileMenuItems.tsx'
import { openProfileDialog } from '../game/profileDialog.ts'
import { useRestoreFocus } from '../game/useRestoreFocus.ts'
import { formatBytes } from '../i18n/bytes.ts'
import { modsLabel, problemsLabel, updatesLabel } from '../i18n/counts.ts'
import { lowerFirst } from '../i18n/lowerFirst.ts'
import { useBadges } from '../mods/badges.ts'
import { openShare } from '../share/store.ts'
import { MenuHeading } from '../shell/TitleMenu.tsx'
import { Tooltip } from '../shell/Tooltip.tsx'
import { space } from '../theme/density.ts'
import { chosenModCount, userModCount } from './count.ts'
import { HealthCheckBadge } from './HealthCheckBadge.tsx'
import { ProfileMark } from './ProfileMark.tsx'
import { useProfiles } from './store.ts'
import { joinSummary, knownCount, originLine } from './summary.ts'

const DRAG_TINT_ALPHA = 0.24
const panelSx = { bgcolor: 'var(--mortar-paper-78)', borderRadius: '6px' }
function RowMenu({
  profile,
  anchor,
  onClose,
  onRename,
  returnFocus,
}: {
  profile: Profile
  anchor: HTMLElement | null
  onClose: () => void
  onRename: () => void
  returnFocus: () => void
}) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  // An action that moves focus itself (rename, delete) turns the return to the ⋯ button off.
  const refocus = useRef(true)
  const choose =
    (run: () => void, keepFocus = true) =>
    () => {
      refocus.current = keepFocus
      onClose()
      run()
    }
  return (
    <Menu
      anchorEl={anchor}
      open={anchor !== null}
      onClose={() => {
        refocus.current = true
        onClose()
      }}
      disableRestoreFocus={true}
      slotProps={{
        transition: {
          onExited: () => {
            if (refocus.current) {
              returnFocus()
            }
          },
        },
      }}
    >
      <MenuHeading>{t`This profile`}</MenuHeading>
      <ProfileMenuItem
        icon={<Pencil size={16} />}
        label={t`Rename`}
        onClick={choose(onRename, false)}
      />
      <ProfileMenuItem
        icon={<Palette size={16} />}
        label={t`Edit profile…`}
        onClick={choose(() => openProfileDialog('edit', profile.id), false)}
      />
      <CoverMenuItems game={game} profile={profile} close={choose(() => undefined)} />
      <MoreMenuItems profile={profile} close={choose(() => undefined)} />
    </Menu>
  )
}

function useRowSummary(profile: Profile): string {
  const { t } = useLingui()
  const mods = userModCount(profile)
  const chosen = chosenModCount(profile)
  const modsText = chosen < mods ? `${modsLabel(mods)} · ${t`${chosen} chosen`}` : modsLabel(mods)
  const badge = useBadges((s) => s.byProfile[profile.id])
  const updatesText = updatesLabel(badge?.updates ?? 0)
  const problemsText = problemsLabel(badge?.problems ?? 0)
  return joinSummary([
    modsText,
    knownCount(badge?.updates, updatesText),
    knownCount(badge?.problems, problemsText),
    originLine(profile.origin, profile.copyOf, {
      link: t`imported from a link`,
      mortar: t`imported from a .mortar file`,
      gameMods: t`imported from the game's Mods folder`,
      copy: (name) => lowerFirst(t`Copy of ${{ name }}`),
    }),
  ])
}

// Where the profile's disk goes: the store items it uses (and how much of that other profiles use too), its
// deployed mods folder, and its own saves when it keeps them.
function BudgetLine({ budget }: { budget: ProfileBudget | undefined }) {
  const { t } = useLingui()
  if (budget === undefined) {
    return null
  }
  const store = formatBytes(budget.store)
  const shared = formatBytes(budget.shared)
  const deployed = formatBytes(budget.deployed)
  const line = joinSummary([
    budget.shared > 0
      ? t`${store} in the store (${shared} shared with other profiles)`
      : t`${store} in the store`,
    t`${deployed} deployed`,
    budget.saves > 0 ? t`${{ size: formatBytes(budget.saves) }} of saves` : undefined,
  ])
  return (
    <Typography
      noWrap={true}
      title={line}
      sx={{ fontSize: 'base.fontSize', color: 'text.secondary' }}
    >
      {line}
    </Typography>
  )
}

export const ProfileRow = memo(function ProfileRowView({
  profile,
  budget,
}: {
  profile: Profile
  budget?: ProfileBudget | undefined
}) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game?.id ?? '')
  const rename = useProfiles((s) => s.rename)
  const [renaming, setRenaming] = useState(false)
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const more = useRef<HTMLButtonElement>(null)
  useRestoreFocus(renaming, more)
  const {
    attributes,
    listeners,
    setNodeRef,
    setActivatorNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: profile.id })
  const summary = useRowSummary(profile)
  return (
    <Box
      ref={setNodeRef}
      style={{
        transform: CSS.Transform.toString(transform && { ...transform, x: 0 }),
        transition,
      }}
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: space.gap,
        height: 'auto',
        minHeight: 64,
        py: 0.75,
        px: space.gap,
        boxSizing: 'border-box',
        mb: '6px',
        ...panelSx,
        border: '2px solid',
        borderColor: isDragging ? 'primary.main' : 'transparent',
        bgcolor: (th) =>
          isDragging ? alpha(th.palette.primary.main, DRAG_TINT_ALPHA) : panelSx.bgcolor,
        position: 'relative',
        zIndex: isDragging ? 1 : 0,
      }}
    >
      <Tooltip title={t`Reorder ${{ name: profile.name }}`}>
        <IconButton
          ref={setActivatorNodeRef}
          aria-label={t`Reorder ${{ name: profile.name }}`}
          {...attributes}
          {...listeners}
          sx={{
            width: 32,
            height: 44,
            borderRadius: '6px',
            color: 'text.secondary',
            cursor: 'grab',
            touchAction: 'none',
          }}
        >
          <GripVertical size={16} />
        </IconButton>
      </Tooltip>
      <Box sx={{ flex: 1, minWidth: 0 }}>
        {renaming ? (
          <NameField
            size="small"
            initial={profile.name}
            label={t`Profile name`}
            onSubmit={(name) => rename(profile.id, name)}
            onCancel={() => setRenaming(false)}
          />
        ) : (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
            <ProfileMark profile={profile} />
            <Typography
              component="h2"
              noWrap={true}
              title={profile.name}
              sx={{ fontSize: 17, fontWeight: 600 }}
            >
              {profile.name}
            </Typography>
            <HealthCheckBadge game={game} profileId={profile.id} />
          </Box>
        )}
        {profile.description ? (
          <Tooltip title={profile.description}>
            <Typography
              noWrap={true}
              tabIndex={0}
              sx={{ fontSize: 'base.fontSize', color: 'text.secondary' }}
            >
              {profile.description}
            </Typography>
          </Tooltip>
        ) : null}
        <Typography
          noWrap={true}
          title={summary}
          sx={{ fontSize: 'base.fontSize', color: 'text.secondary' }}
        >
          {summary}
        </Typography>
        <BudgetLine budget={budget} />
      </Box>
      <Button
        variant="outlined"
        color="inherit"
        startIcon={<Share2 size={15} />}
        aria-label={t`Share ${{ name: profile.name }}`}
        onClick={() => openShare(profile.id)}
        sx={{ height: 40 }}
      >
        {t`Share`}
      </Button>
      <Tooltip title={t`Actions for ${{ name: profile.name }}`}>
        <IconButton
          ref={more}
          data-actions={profile.id}
          aria-label={t`Actions for ${{ name: profile.name }}`}
          aria-haspopup="menu"
          aria-expanded={anchor !== null}
          onClick={(e) => setAnchor(e.currentTarget)}
          sx={{
            width: 40,
            height: 40,
            borderRadius: '6px',
            bgcolor: anchor ? 'var(--mortar-hairline)' : 'transparent',
          }}
        >
          <MoreHorizontal size={18} />
        </IconButton>
      </Tooltip>
      <RowMenu
        profile={profile}
        anchor={anchor}
        onClose={() => setAnchor(null)}
        onRename={() => setRenaming(true)}
        returnFocus={() => more.current?.focus()}
      />
    </Box>
  )
})
