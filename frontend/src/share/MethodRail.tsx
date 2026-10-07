import { useLingui } from '@lingui/react/macro'
import { Box, Button } from '@mui/material'
import { FileJson, List, Package, Save, Send, Share2 } from 'lucide-react'
import type { ReactNode } from 'react'
import type { MethodEntry, ShareMethod } from './methods.ts'

export function MethodRail({
  method,
  setMethod,
  methods,
}: {
  method: ShareMethod
  setMethod: (next: ShareMethod) => void
  methods: MethodEntry[]
}) {
  const { t } = useLingui()
  const rail: Record<ShareMethod, { icon: ReactNode; name: string; hint: string }> = {
    link: {
      icon: <Share2 size={18} />,
      name: t`Link`,
      hint: t`Paste anywhere; they open it in Mortar`,
    },
    file: {
      icon: <Save size={18} />,
      name: t`File`,
      hint: t`.mortar, with your mod settings`,
    },
    list: {
      icon: <List size={18} />,
      name: t`Mod list`,
      hint: t`Text for Discord or a forum`,
    },
    nexus: {
      icon: <FileJson size={18} />,
      name: t`Nexus collection`,
      hint: t`A draft on Nexus Mods`,
    },
    nearby: {
      icon: <Send size={18} />,
      name: t`Nearby computer`,
      hint: t`Send over your network`,
    },
    thunderstore: {
      icon: <Package size={18} />,
      name: t`r2modman`,
      hint: t`A code or a Thunderstore modpack`,
    },
  }
  return (
    <Box
      role="tablist"
      aria-orientation="vertical"
      aria-label={t`How to share`}
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: 0.5,
        p: '16px 12px',
        overflowY: 'auto',
        borderRight: '1px solid var(--mortar-hairline-muted)',
      }}
    >
      {methods.map((m) => {
        const selected = m.id === method
        const entry = rail[m.id]
        return (
          <Button
            key={m.id}
            role="tab"
            aria-selected={selected}
            color="inherit"
            disabled={m.disabled}
            onClick={() => setMethod(m.id)}
            sx={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'flex-start',
              gap: 1.5,
              p: '10px 12px',
              textAlign: 'left',
              textTransform: 'none',
              borderRadius: '8px',
              border: '1px solid',
              borderColor: selected ? 'primary.main' : 'transparent',
              bgcolor: selected ? 'var(--mortar-hairline-14)' : 'transparent',
            }}
          >
            {entry.icon}
            <Box sx={{ display: 'flex', flexDirection: 'column', minWidth: 0 }}>
              <Box component="span" sx={{ fontWeight: 600, fontSize: 14 }}>
                {entry.name}
              </Box>
              <Box component="span" sx={{ fontSize: 12, color: 'text.secondary' }}>
                {entry.hint}
              </Box>
            </Box>
          </Button>
        )
      })}
    </Box>
  )
}
