import { useLingui } from '@lingui/react/macro'
import { Box, Dialog, IconButton, Typography } from '@mui/material'
import { ArrowLeft, X } from 'lucide-react'
import { type ReactNode, useId } from 'react'

export type DialogSize = 'sm' | 'md' | 'lg' | 'xl'
export type DialogTone = 'neutral' | 'warning' | 'danger'

const WIDTH: Record<DialogSize, number> = { sm: 480, md: 560, lg: 760, xl: 1180 }
const TONE_COLOR: Record<DialogTone, string> = {
  neutral: 'var(--mortar-accent-ink)',
  warning: 'warning.main',
  danger: 'error.main',
}
const PAD_X = 3
const ICON_TILE_PX = 44
const TITLE_PX = 17

// A square tile behind the dialog's icon, tinted with the tone's colour.
export function ToneIcon({ tone, children }: { tone: DialogTone; children: ReactNode }) {
  return (
    <Box
      aria-hidden={true}
      sx={{
        width: ICON_TILE_PX,
        height: ICON_TILE_PX,
        flexShrink: 0,
        borderRadius: '10px',
        display: 'grid',
        placeItems: 'center',
        color: TONE_COLOR[tone],
        bgcolor: 'color-mix(in srgb, currentColor 16%, transparent)',
      }}
    >
      {children}
    </Box>
  )
}

/** The footer row: something to say on the left (a reason, a note), the buttons on the right, Cancel before the action. */
export function DialogFooter({ aside, children }: { aside?: ReactNode; children: ReactNode }) {
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        px: PAD_X,
        py: 1.75,
        borderTop: '1px solid var(--mortar-hairline-muted)',
        bgcolor: 'var(--mortar-raised)',
      }}
    >
      <Box sx={{ flex: 1, minWidth: 0, fontSize: 'small.fontSize', color: 'text.secondary' }}>
        {aside}
      </Box>
      {children}
    </Box>
  )
}

/**
 * Every dialog's frame: a header with an optional icon, the title and a line under it, and a close button; a body
 * that scrolls on its own; and a footer that stays put. Esc and the close button both call onClose.
 */
export function AppDialog({
  open,
  onClose,
  title,
  subtitle,
  icon,
  size = 'md',
  footer,
  closable = true,
  closeButton = footer === undefined || footer === null,
  onBack,
  height,
  flush = false,
  tabs,
  children,
}: {
  open: boolean
  onClose: () => void
  title: ReactNode
  subtitle?: ReactNode
  icon?: ReactNode
  size?: DialogSize
  footer?: ReactNode
  // False for a dialog that wants an answer: no X and clicks outside do nothing; Esc still means Cancel (onClose).
  closable?: boolean
  // An X in the header; by default only a dialog without a footer has one, since a footer holds its own Cancel or Close.
  closeButton?: boolean
  // A step back inside the dialog: an arrow before the title, and Esc goes back before it closes.
  onBack?: (() => void) | undefined
  // A fixed height, for a body whose content changes size as the reader moves through it.
  height?: string | number | undefined
  // The body draws its own edges (a grid or list running to the dialog's sides).
  flush?: boolean
  // Tabs that switch the body, attached under the header and drawn as its dividing line.
  tabs?: ReactNode
  children?: ReactNode
}) {
  const { t } = useLingui()
  const titleId = useId()
  return (
    <Dialog
      open={open}
      onClose={(_, reason) => {
        if (reason === 'escapeKeyDown' && onBack) {
          onBack()
        } else if (closable || reason === 'escapeKeyDown') {
          onClose()
        }
      }}
      maxWidth={false}
      aria-labelledby={titleId}
      slotProps={{
        paper: {
          sx: {
            width: WIDTH[size],
            ...(height === undefined ? {} : { height }),
            maxWidth: 'calc(100% - 32px)',
            maxHeight: 'calc(100% - 64px)',
            m: 2,
            borderRadius: '10px',
            border: '1px solid var(--mortar-hairline)',
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden',
          },
        },
      }}
    >
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1.75,
          px: PAD_X,
          py: 2,
          // The dialog's own header stands apart from any headings in its body.
          borderBottom: children && !tabs ? '1px solid var(--mortar-hairline-muted)' : 'none',
        }}
      >
        {onBack ? (
          <IconButton aria-label={t`Back`} onClick={onBack} size="small" sx={{ ml: -1 }}>
            <ArrowLeft size={18} />
          </IconButton>
        ) : null}
        {icon}
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Typography
            id={titleId}
            component="h2"
            sx={{ fontSize: TITLE_PX, fontWeight: 600, lineHeight: 1.25 }}
          >
            {title}
          </Typography>
          {subtitle ? (
            <Typography
              sx={{ fontSize: 'medium.fontSize', color: 'text.secondary', lineHeight: 1.35 }}
            >
              {subtitle}
            </Typography>
          ) : null}
        </Box>
        {closable && closeButton ? (
          <IconButton aria-label={t`Close`} onClick={onClose} size="small" sx={{ mr: -1 }}>
            <X size={18} />
          </IconButton>
        ) : null}
      </Box>
      {tabs ? (
        <Box sx={{ px: PAD_X, borderBottom: '1px solid var(--mortar-hairline-muted)' }}>{tabs}</Box>
      ) : null}
      {children ? (
        <Box
          sx={{
            ...(flush ? { display: 'flex', flexDirection: 'column' } : { px: PAD_X, py: 2.5 }),
            overflowY: 'auto',
            flex: '1 1 auto',
            minHeight: 0,
            // The body's own padding is the gap above the footer; a last child's margin would add to it.
            '& > :last-child': { mb: 0 },
          }}
        >
          {children}
        </Box>
      ) : null}
      {footer}
    </Dialog>
  )
}
