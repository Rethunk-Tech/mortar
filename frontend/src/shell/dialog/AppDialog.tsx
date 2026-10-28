import { useLingui } from '@lingui/react/macro'
import { Box, Dialog, IconButton } from '@mui/material'
import { ArrowLeft, X } from 'lucide-react'
import { type ReactNode, useId } from 'react'
import { space } from '../../theme/density.ts'
import { paperSx } from './frame.ts'

export type DialogSize = 'sm' | 'md' | 'lg' | 'xl' | 'full'
export type DialogTone = 'neutral' | 'warning' | 'danger'

const WIDTH: Record<Exclude<DialogSize, 'full'>, number> = { sm: 480, md: 560, lg: 760, xl: 1180 }
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
    <div className="app-dialog-footer">
      <div className="app-dialog-aside">{aside}</div>
      {children}
    </div>
  )
}

// The frame's parts are plain elements styled from the paper: a dialog's frame redraws whenever anything in it changes,
// and each Box or Typography would be three components more each time.
const frameSx = {
  ...paperSx,
  '& > .app-dialog-header': {
    display: 'flex',
    alignItems: 'center',
    gap: 1.75,
    px: PAD_X,
    py: space.pad,
    // The dialog's own header stands apart from any headings in its body.
    '&.app-dialog-ruled': { borderBottom: '1px solid var(--mortar-hairline-muted)' },
  },
  '& .app-dialog-heading': { flex: 1, minWidth: 0 },
  '& .app-dialog-title': { m: 0, fontSize: TITLE_PX, fontWeight: 600, lineHeight: 1.25 },
  '& .app-dialog-subtitle': {
    m: 0,
    fontSize: 'medium.fontSize',
    color: 'text.secondary',
    lineHeight: 1.35,
  },
  '& > .app-dialog-tabs': { px: PAD_X, borderBottom: '1px solid var(--mortar-hairline-muted)' },
  '& > .app-dialog-body': {
    px: PAD_X,
    // The form's gap between fields, so the last field sits as far from the footer as from the field above it.
    py: 3,
    overflowY: 'auto',
    flex: '1 1 auto',
    minHeight: 0,
    // The body draws its own edges (a grid or list running to the dialog's sides).
    '&.app-dialog-flush': { display: 'flex', flexDirection: 'column', p: 0 },
    // The body's own padding is the gap above the footer; a last child's margin would add to it.
    '& > :last-child': { mb: 0 },
    // A switch, checkbox or radio sits on the fields' left edge, as MUI draws it; a row's own margin reset would indent it.
    '& .MuiFormControlLabel-root': { ml: '-11px' },
    // MUI pads each radio 9px; a radio list's first and last rows sit on its text, as a field's other controls do.
    '& .MuiRadioGroup-root': { my: '-9px' },
  },
  '& .app-dialog-footer': {
    display: 'flex',
    alignItems: 'center',
    gap: space.gap,
    px: PAD_X,
    py: 1.75,
    borderTop: '1px solid var(--mortar-hairline-muted)',
    bgcolor: 'var(--mortar-raised)',
  },
  '& .app-dialog-aside': {
    flex: 1,
    minWidth: 0,
    fontSize: 'small.fontSize',
    color: 'text.secondary',
  },
  // FormDialog's form and the reason its submit button waits.
  '& .app-dialog-form': { display: 'flex', flexDirection: 'column', gap: 3 },
  '& .app-dialog-issue': { display: 'inline-flex', alignItems: 'center', gap: 0.75 },
} as const

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
  actions,
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
  // Controls in the header row between the title and the close button.
  actions?: ReactNode
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
      fullScreen={size === 'full'}
      aria-labelledby={titleId}
      slotProps={{
        paper: {
          sx: [
            frameSx,
            size === 'full' ? {} : { width: WIDTH[size] },
            height === undefined ? {} : { height },
          ],
        },
      }}
    >
      <div
        className={children && !tabs ? 'app-dialog-header app-dialog-ruled' : 'app-dialog-header'}
      >
        {onBack ? (
          <IconButton aria-label={t`Back`} onClick={onBack} size="small" sx={{ ml: -1 }}>
            <ArrowLeft size={18} />
          </IconButton>
        ) : null}
        {icon}
        <div className="app-dialog-heading">
          <h2 id={titleId} className="app-dialog-title">
            {title}
          </h2>
          {subtitle ? <p className="app-dialog-subtitle">{subtitle}</p> : null}
        </div>
        {actions}
        {closable && closeButton ? (
          <IconButton aria-label={t`Close`} onClick={onClose} size="small" sx={{ mr: -1 }}>
            <X size={18} />
          </IconButton>
        ) : null}
      </div>
      {tabs ? <div className="app-dialog-tabs">{tabs}</div> : null}
      {children ? (
        <div className={flush ? 'app-dialog-body app-dialog-flush' : 'app-dialog-body'}>
          {children}
        </div>
      ) : null}
      {footer}
    </Dialog>
  )
}
