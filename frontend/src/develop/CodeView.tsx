import { useLingui } from '@lingui/react/macro'
import { Box, IconButton } from '@mui/material'
import type { Theme } from '@mui/material/styles'
import { useVirtualizer } from '@tanstack/react-virtual'
import { Copy } from 'lucide-react'
import {
  type KeyboardEvent,
  memo,
  type RefObject,
  useCallback,
  useEffect,
  useMemo,
  useRef,
} from 'react'
import { copyText } from '../share/copyText.ts'
import { Tooltip } from '../shell/Tooltip.tsx'
import { MONO } from '../theme/theme.ts'
import { type TokenKind, tokenize } from './codeTokens.ts'
import { type SymbolClick, symbolAt } from './symbolAt.ts'

const LINE_PX = 20
const OVERSCAN = 30
// Frames a jump keeps correcting itself while the rows it passes are measured.
const JUMP_TRIES = 8

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

// Rows are measured as they draw, and long ones wrap, so a jump far down lands short of an estimate: it repeats each
// frame until the row is on screen, and does nothing when it already is. The returned function cancels it.
function useJump(
  virtualizer: { scrollToIndex: (index: number, options: { align: 'start' | 'center' }) => void },
  scroller: RefObject<HTMLDivElement | null>,
) {
  return useCallback(
    (index: number, align: 'start' | 'center') => {
      let tries = 0
      let frame = 0
      const shown = () => {
        const box = scroller.current?.getBoundingClientRect()
        const row = scroller.current
          ?.querySelector(`[data-index="${index}"]`)
          ?.getBoundingClientRect()
        return (
          box !== undefined && row !== undefined && row.top >= box.top && row.bottom <= box.bottom
        )
      }
      // A row already on screen stays where it is, so the side being read does not move under the reader.
      const step = () => {
        if (shown()) {
          return
        }
        virtualizer.scrollToIndex(index, { align })
        tries += 1
        if (tries < JUMP_TRIES) {
          frame = requestAnimationFrame(step)
        }
      }
      step()
      return () => cancelAnimationFrame(frame)
    },
    [virtualizer, scroller],
  )
}

/** Decompiled code with line numbers and highlighting, drawn a screenful at a time; Copy takes all of it. `line` is
 * scrolled to the top when it changes, `follow` into view each time a new one is passed; `marked` lines are tinted, `onPick`
 * hears a line clicked and `onSymbol` a name right-clicked, with where. */
export function CodeView({
  source,
  il,
  line,
  follow = null,
  marked = NO_MARKS,
  onPick,
  onSymbol,
}: {
  source: string
  il: boolean
  line: number
  follow?: { line: number } | null
  marked?: ReadonlySet<number>
  onPick?: (line: number) => void
  onSymbol?: ((at: SymbolClick) => void) | undefined
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
  const jump = useJump(virtualizer, scroller)
  useEffect(() => (line >= 0 ? jump(line, 'start') : undefined), [line, jump])
  useEffect(() => (follow ? jump(follow.line, 'center') : undefined), [follow, jump])
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
            width: '100%',
          }}
        >
          {virtualizer.getVirtualItems().map((v) => (
            <Box
              key={v.key}
              // Long lines wrap, so each row is measured rather than assumed one line tall.
              ref={virtualizer.measureElement}
              data-index={v.index}
              onClick={onPick ? () => onPick(v.index) : undefined}
              onContextMenu={(e) => {
                // With text selected, the right-click keeps the menu that copies it.
                const name =
                  onSymbol && !globalThis.getSelection()?.toString()
                    ? symbolAt(e.clientX, e.clientY)
                    : ''
                if (name !== '') {
                  e.preventDefault()
                  onSymbol?.({
                    name,
                    x: e.clientX,
                    y: e.clientY,
                    line: v.index,
                    text: lines[v.index] ?? '',
                    il,
                  })
                }
              }}
              {...(onPick
                ? {
                    role: 'button',
                    tabIndex: 0,
                    onKeyDown: (e: KeyboardEvent<HTMLElement>) => {
                      if (e.target === e.currentTarget && (e.key === 'Enter' || e.key === ' ')) {
                        e.preventDefault()
                        onPick(v.index)
                      }
                    },
                  }
                : {})}
              sx={{
                cursor: onPick ? 'pointer' : undefined,
                '&:focus-visible': {
                  outline: '2px solid',
                  outlineColor: 'primary.main',
                  outlineOffset: -2,
                },
                bgcolor: marked.has(v.index) ? MARK_BG : undefined,
                position: 'absolute',
                top: 0,
                left: 0,
                width: '100%',
                lineHeight: `${LINE_PX}px`,
                transform: `translateY(${v.start}px)`,
                display: 'flex',
                whiteSpace: 'pre-wrap',
                overflowWrap: 'anywhere',
                tabSize: 4,
              }}
            >
              <Box
                component="span"
                aria-hidden={true}
                sx={{
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
              <Box component="span" sx={{ pl: 1.5, pr: 2, flex: 1, minWidth: 0 }}>
                <Line text={lines[v.index] ?? ''} il={il} />
              </Box>
            </Box>
          ))}
        </Box>
      </Box>
    </Box>
  )
}
