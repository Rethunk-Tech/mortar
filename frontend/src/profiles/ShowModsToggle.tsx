import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { ChevronDown, ChevronRight } from 'lucide-react'

/** Show mods / Hide mods under a summary line; the list it opens is the caller's. */
export function ShowModsToggle({ open, onToggle }: { open: boolean; onToggle: () => void }) {
  const { t } = useLingui()
  return (
    <Button
      size="small"
      aria-expanded={open}
      onClick={onToggle}
      startIcon={open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
    >
      {open ? t`Hide mods` : t`Show mods`}
    </Button>
  )
}
