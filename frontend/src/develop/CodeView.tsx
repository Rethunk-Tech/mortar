import { useLingui } from '@lingui/react/macro'
import { Box, IconButton } from '@mui/material'
import type { Theme } from '@mui/material/styles'
import { useVirtualizer } from '@tanstack/react-virtual'
import { Copy } from 'lucide-react'
import { memo, useEffect, useMemo, useRef } from 'react'
import { copyText } from '../share/copyText.ts'
import { Tooltip } from '../shell/Tooltip.tsx'
import { MONO } from '../theme/theme.ts'
import { type TokenKind, tokenize } from './codeTokens.ts'

const LINE_PX = 20
const OVERSCAN = 30

// Palette colours per run: the light shade on a dark theme, the dark shade on a light one.
const PALETTE: Partial<
  Record<TokenKind, 'info' | 'success' | 'warning' | 'secondary' | 'primary'>
> = {
  keyword: 'info',
  type: 'success',
  string: 'warning',
  number: 'secondary',
  directive: 'primary',
}

function colour(kind: TokenKind) {
  if (kind === 'comment' || kind === 'label') {
    return 'text.secondary'
  }
  const key = PALETTE[kind]
  if (!key) {
    return 'text.primary'
  }
  return (t: Theme) => (t.palette.mode === 'dark' ? t.palette[key].light : t.palette[key].dark)
}

const Line = memo(function LineView({ text, il }: { text: string; il: boolean }) {
  return (
    <>
      {tokenize(text, il).map((tok) => (
        <Box
          key={tok.start}
          component="span"
          sx={{ color: colour(tok.kind), fontStyle: tok.kind === 'comment' ? 'italic' : undefined }}
        >
          {tok.text}
        </Box>
      ))}
    </>
  )
})

const NO_MARKS: ReadonlySet<number> = new Set()
const MARK_BG = 'color-mix(in srgb, var(--mortar-accent-ink) 18%, transparent)'

/** Decompiled code with line numbers and highlighting, drawn a screenful at a time; Copy takes all of it. `line` is
 * scrolled to the top when it changes, `follow` into view; `marked` lines are tinted and `onPick` hears a line clicked. */
export function CodeView({
  source,
  il,
  line,
  follow = -1,
  marked = NO_MARKS,
  onPick,
}: {
  source: string
  il: boolean
  line: number
  follow?: number
  marked?: ReadonlySet<number>
  onPick?: (line: number) => void
}) {
  const { t } = useLingui()
  const scroller = useRef<HTMLDivElement | null>(null)
  const lines = useMemo(() => source.split('\n'), [source])
  const gutter = `${String(lines.length).length + 2}ch`
  const virtualizer = useVirtualizer({
    count: lines.length,
    getScrollElement: () => scroller.current,
    estimateSize: () => LINE_PX,
    overscan: OVERSCAN,
  })
  useEffect(() => {
    if (line >= 0) {
      virtualizer.scrollToIndex(line, { align: 'start' })
    }
  }, [line, virtualizer])
  useEffect(() => {
    if (follow >= 0) {
      virtualizer.scrollToIndex(follow, { align: 'center' })
    }
  }, [follow, virtualizer])
  return (
    <Box sx={{ position: 'relative', flex: 1, minHeight: 0, display: 'flex' }}>
      <Tooltip title={t`Copy code`}>
        <IconButton
          size="small"
          aria-label={t`Copy code`}
          onClick={() => copyText(source, t`Code copied`)}
          sx={{
            position: 'absolute',
            top: 8,
            right: 20,
            zIndex: 2,
            bgcolor: 'var(--mortar-raised)',
          }}
        >
          <Copy size={14} />
        </IconButton>
      </Tooltip>
      <Box
        ref={scroller}
        role="region"
        aria-label={t`Code`}
        sx={{
          flex: 1,
          overflow: 'auto',
          fontFamily: MONO,
          fontSize: 'small.fontSize',
          userSelect: 'text',
        }}
      >
        <Box
          sx={{
            position: 'relative',
            height: virtualizer.getTotalSize(),
            minWidth: '100%',
            width: 'max-content',
          }}
        >
          {virtualizer.getVirtualItems().map((v) => (
            <Box
              key={v.key}
              onClick={onPick ? () => onPick(v.index) : undefined}
              sx={{
                cursor: onPick ? 'pointer' : undefined,
                bgcolor: marked.has(v.index) ? MARK_BG : undefined,
                position: 'absolute',
                top: 0,
                left: 0,
                height: LINE_PX,
                lineHeight: `${LINE_PX}px`,
                transform: `translateY(${v.start}px)`,
                display: 'flex',
                whiteSpace: 'pre',
                tabSize: 4,
              }}
            >
              <Box
                component="span"
                aria-hidden={true}
                sx={{
                  position: 'sticky',
                  left: 0,
                  width: gutter,
                  flexShrink: 0,
                  pr: 1.5,
                  textAlign: 'right',
                  color: 'text.disabled',
                  bgcolor: 'var(--mortar-raised)',
                  userSelect: 'none',
                }}
              >
                {v.index + 1}
              </Box>
              <Box component="span" sx={{ pl: 1.5, pr: 2 }}>
                <Line text={lines[v.index] ?? ''} il={il} />
              </Box>
            </Box>
          ))}
        </Box>
      </Box>
    </Box>
  )
}
