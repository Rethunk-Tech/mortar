import { useLingui } from '@lingui/react/macro'
import { Button, Menu, MenuItem } from '@mui/material'
import { ChevronDown } from 'lucide-react'
import { useState } from 'react'
import { RememberSettingChoice } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/problems/service.ts'
import { SetConfigValue } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { useProfiles } from '../../profiles/store.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import type { Problem } from '../lookup.ts'
import { useMods } from '../store.ts'
import { useLocked } from '../useLocked.ts'

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
  const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null)
  const values = setting.suggested ?? []
  const apply = async (value: string) => {
    setAnchorEl(null)
    const { game, openId } = useProfiles.getState()
    if (!(game && openId)) {
      return
    }
    try {
      await SetConfigValue(game.id, openId, setting.key, setting.uniqueId, setting.field, value)
      await RememberSettingChoice(game.id, openId, setting.uniqueId, setting.field, value)
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
        <Button
          size="small"
          variant="contained"
          color="warning"
          disabled={locked}
          onClick={() => apply(first)}
          sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
        >
          {label(first)}
        </Button>
      ) : (
        <Button
          size="small"
          color="inherit"
          variant="text"
          onClick={() =>
            useMods.getState().restoreDismissed(dismissedToken).catch(reportUnexpected)
          }
          sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
        >
          {t`Restore`}
        </Button>
      )}
      {values.length > 1 ? (
        <>
          <Button
            size="small"
            variant="contained"
            color="warning"
            aria-label={t`More setting values`}
            disabled={locked}
            onClick={(event) => setAnchorEl(event.currentTarget)}
            sx={{ minWidth: 28, width: 28, height: 28, px: 0, flexShrink: 0 }}
          >
            <ChevronDown size={15} aria-hidden={true} />
          </Button>
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
        sx={{ height: 28, whiteSpace: 'nowrap', flexShrink: 0 }}
      >
        {t`Dismiss`}
      </Button>
    </>
  )
}
