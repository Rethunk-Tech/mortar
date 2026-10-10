import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Menu, Typography } from '@mui/material'
import { ChevronDown, Info, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { LinkedText } from './ModLinks.tsx'
import { useMods } from './store.ts'

interface CleanupItem {
  key: string
  id: string
  name: string
  reason?: string
  text?: string
  choices?: { key: string; name: string }[]
  // A finding the user may judge wrong offers Dismiss; one already dismissed offers only Restore.
  onDismiss?: () => void
  onRestore?: () => void
}

// RemoveOne asks which mod of a group goes, since any one of them may be the one to keep.
function RemoveOne({ choices }: { choices: { key: string; name: string }[] }) {
  const { t } = useLingui()
  const remove = useMods((s) => s.remove)
  const mods = useMods((s) => s.mods)
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  return (
    <>
      <Button
        size="small"
        endIcon={<ChevronDown size={14} />}
        aria-haspopup="menu"
        aria-expanded={anchor !== null}
        onClick={(e) => setAnchor(e.currentTarget)}
      >
        {t`Remove one…`}
      </Button>
      <Menu anchorEl={anchor} open={anchor !== null} onClose={() => setAnchor(null)}>
        {choices.map((choice) => {
          const mod = mods.find((m) => m.key === choice.key)
          return (
            <MenuAction
              key={choice.key}
              tone="error"
              disabled={mod === undefined}
              icon={<Trash2 size={16} />}
              label={t`Remove ${{ name: choice.name }}`}
              onClick={() => {
                setAnchor(null)
                if (mod !== undefined) {
                  remove(mod).catch(reportUnexpected)
                }
              }}
            />
          )
        })}
      </Menu>
    </>
  )
}

function CleanupRow({ cleanup }: { cleanup: CleanupItem }) {
  const { t } = useLingui()
  const remove = useMods((s) => s.remove)
  const mod = useMods((s) => s.mods.find((candidate) => candidate.key === cleanup.key))
  const who = cleanup.name.trim() === '' ? t`Unknown mod` : cleanup.name
  const reason = cleanup.reason || t`Not needed by any enabled mod`
  const sentence = cleanup.text ?? t`${who}: ${reason}`
  const removal = cleanup.choices ? (
    <RemoveOne choices={cleanup.choices} />
  ) : (
    <DisabledReason title={t`This mod is no longer in the profile.`} disabled={mod === undefined}>
      <Button
        size="small"
        aria-label={t`Remove ${{ name: who }}`}
        disabled={mod === undefined}
        onClick={() => {
          if (mod !== undefined) {
            remove(mod).catch(reportUnexpected)
          }
        }}
      >
        {t`Remove`}
      </Button>
    </DisabledReason>
  )
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'flex-start',
        gap: space.gap,
        flexShrink: 0,
        pl: space.pad,
        pr: 0.75,
        py: space.gap,
        fontSize: 14,
        bgcolor: 'var(--mortar-overlay-45)',
        borderRadius: '6px',
        ...(cleanup.onRestore ? { opacity: 0.75 } : {}),
      }}
    >
      <Box sx={{ display: 'flex', color: 'text.secondary', mt: '2px' }}>
        <Info size={16} aria-hidden={true} />
      </Box>
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography
          title={cleanup.name.trim() === '' ? cleanup.id : undefined}
          sx={{ fontSize: 14, whiteSpace: 'normal', wordBreak: 'break-word' }}
        >
          <LinkedText
            text={sentence}
            links={[{ name: cleanup.name, key: cleanup.key, id: cleanup.id }]}
          />
        </Typography>
      </Box>
      {cleanup.onDismiss ? (
        <Button
          size="small"
          color="inherit"
          aria-label={t`Dismiss ${{ name: sentence }}`}
          onClick={cleanup.onDismiss}
        >
          {t`Dismiss`}
        </Button>
      ) : null}
      {cleanup.onRestore ? (
        <Button
          size="small"
          aria-label={t`Restore ${{ label: sentence }}`}
          onClick={cleanup.onRestore}
        >
          {t`Restore`}
        </Button>
      ) : null}
      {cleanup.onRestore === undefined ? removal : null}
    </Box>
  )
}

function removeCleanup(
  cleanup: { key: string }[],
  removeMany: ReturnType<typeof useMods.getState>['removeMany'],
) {
  const mods = cleanup
    .map((item) => useMods.getState().mods.find((mod) => mod.key === item.key))
    .filter((mod): mod is NonNullable<typeof mod> => mod !== undefined)
  removeMany(mods).catch(reportUnexpected)
}

// Remove all is offered only where every row is a sure removal.
export function CleanupSection({
  cleanup,
  removeAll = true,
}: {
  cleanup: CleanupItem[]
  removeAll?: boolean
}) {
  const { t } = useLingui()
  const removeMany = useMods((s) => s.removeMany)
  const [confirmCleanup, setConfirmCleanup] = useState(false)
  if (cleanup.length === 0) {
    return null
  }
  return (
    <Box>
      {removeAll && cleanup.length > 1 ? (
        <Button size="small" sx={{ mb: 1, height: 26 }} onClick={() => setConfirmCleanup(true)}>
          {t`Remove all`}
        </Button>
      ) : null}
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: space.gap }}>
        {cleanup.map((item) => (
          <CleanupRow key={item.key} cleanup={item} />
        ))}
      </Box>
      <ConfirmDialog
        open={confirmCleanup}
        title={t`${plural(cleanup.length, { one: 'Remove all # mod from this profile?', other: 'Remove all # mods from this profile?' })}`}
        body={t`This change can be undone from History.`}
        confirmLabel={t`Remove all`}
        color="error"
        onCancel={() => setConfirmCleanup(false)}
        onConfirm={() => {
          setConfirmCleanup(false)
          removeCleanup(cleanup, removeMany)
        }}
      />
    </Box>
  )
}
