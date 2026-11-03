import { useLingui } from '@lingui/react/macro'
import { Box, Button, Collapse, styled } from '@mui/material'
import { ChevronDown, ChevronUp } from 'lucide-react'
import { useState } from 'react'

const chartSx = {
  display: 'grid',
  gridTemplateColumns: '1fr 1fr',
  gap: 2,
  mt: 1,
  fontSize: 'small.fontSize',
  color: 'text.secondary',
  '& h4': { m: 0, mb: 0.5, fontSize: 'small.fontSize', fontWeight: 600, color: 'text.primary' },
  '& ul': { m: 0, pl: 2 },
} as const
const ChartRoot = styled('div')({})

/** What others may and may not do with the mod's code under the chosen licence, side by side, shown when asked. */
export function LicenseChart({ license }: { license: string }) {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  const useShare = t`Use, change and share it`
  const closed = t`Sell it or put it in a closed mod`
  const notices = t`Remove your copyright notice`
  const rows: Record<string, [string[], string[]]> = {
    '': [
      [useShare, closed],
      [notices, t`Hold you liable`],
    ],
    'apache-2.0': [
      [useShare, closed, t`Use your patents in it`],
      [notices, t`Hide which files they changed`, t`Use your name or trademarks`],
    ],
    'gpl-3.0': [
      [useShare, t`Sell it`],
      [
        t`Share a changed or built-on mod without its source`,
        t`Release that mod under another licence`,
      ],
    ],
    'mpl-2.0': [
      [useShare, t`Combine it with closed code`],
      [t`Keep their changes to your files closed`, notices],
    ],
    unlicense: [[t`Anything at all`], []],
  }
  const [may, mayNot] = rows[license] ?? rows[''] ?? [[], []]
  return (
    <Box sx={{ mt: -2 }}>
      <Button
        size="small"
        aria-expanded={open}
        startIcon={open ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
        onClick={() => setOpen(!open)}
      >
        {t`What this allows`}
      </Button>
      <Collapse in={open} unmountOnExit={true}>
        <ChartRoot sx={chartSx}>
          <div>
            <h4>{t`Others may`}</h4>
            <ul>
              {may.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </div>
          <div>
            <h4>{t`Others may not`}</h4>
            {mayNot.length > 0 ? (
              <ul>
                {mayNot.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            ) : (
              t`Nothing`
            )}
          </div>
        </ChartRoot>
      </Collapse>
    </Box>
  )
}
