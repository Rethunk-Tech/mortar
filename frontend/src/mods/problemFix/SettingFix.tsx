import { useLingui } from '@lingui/react/macro'
import { Button, Menu, MenuItem } from '@mui/material'
import { ChevronDown } from 'lucide-react'
import { useState } from 'react'
import { RememberSettingChoice } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import { SetConfigValue } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { DisabledReason } from '../../shell/DisabledReason.tsx'
import { reportUnexpected } from '../../toasts/report.ts'
import type { Problem } from '../lookup.ts'
import { useMods } from '../store.ts'
import { openTarget } from '../storeView.ts'
import { useLocked } from '../useLocked.ts'
import { RestoreButton } from './RestoreButton.tsx'

export function SettingFix({
  problem,
  dismissedToken,
}: {
  problem: Extract<Problem, { kind: 'setting' }>
  dismissedToken?: string | undefined
}) {
  const { t } = useLingui()
  const { setting } = problem
  const loadProblems = useMods((s) => s.loadProblems)
  const dismissSetting = useMods((s) => s.dismissSetting)
  const locked = useLocked()
  const lockedTitle = t`Stop the game to change mods.`
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)
  const values = setting.suggested ?? []
  const apply = async (value: string) => {
    setAnchorEl(null)
    const at = openTarget()
    if (!at) {
      return
    }
    try {
      await SetConfigValue(at.game, at.id, setting.key, setting.uniqueId, setting.field, value)
      await RememberSettingChoice(at.game, at.id, setting.uniqueId, setting.field, value)
      await loadProblems()
    } catch (error) {
      reportUnexpected(error)
    }
  }
  if (values.length === 0) {
    return null
  }
  const first = values[0] ?? ''
  const label = (value: string) => (value === '' ? t`Set to automatic` : t`Set to ${value}`)
  return (
    <>
      {dismissedToken === undefined ? (
        <DisabledReason title={lockedTitle} disabled={locked}>
          <Button
            size="small"
            variant="contained"
            color="warning"
            disabled={locked}
            onClick={() => apply(first)}
            sx={{ flexShrink: 0 }}
          >
            {label(first)}
          </Button>
        </DisabledReason>
      ) : (
        <RestoreButton token={dismissedToken} />
      )}
      {values.length > 1 ? (
        <>
          <DisabledReason title={lockedTitle} disabled={locked}>
            <Button
              size="small"
              variant="contained"
              color="warning"
              aria-label={t`More setting values`}
              disabled={locked}
              aria-haspopup="menu"
              aria-expanded={anchorEl !== null}
              onClick={(event) => setAnchorEl(event.currentTarget)}
              sx={{ minWidth: 28, width: 28, height: 28, px: 0, flexShrink: 0 }}
            >
              <ChevronDown size={15} aria-hidden={true} />
            </Button>
          </DisabledReason>
          <Menu anchorEl={anchorEl} open={Boolean(anchorEl)} onClose={() => setAnchorEl(null)}>
            {values.map((value) => (
              <MenuItem key={value} onClick={() => apply(value)}>
                {label(value)}
              </MenuItem>
            ))}
          </Menu>
        </>
      ) : null}
      <Button
        size="small"
        color="inherit"
        variant="text"
        onClick={() => dismissSetting(setting).catch(reportUnexpected)}
        sx={{ flexShrink: 0 }}
      >
        {t`Dismiss`}
      </Button>
    </>
  )
}
