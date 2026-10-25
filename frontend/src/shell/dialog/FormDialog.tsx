import { useLingui } from '@lingui/react/macro'
import { Box, Button, CircularProgress, TextField, Typography } from '@mui/material'
import { CircleAlert } from 'lucide-react'
import { type ReactNode, useId } from 'react'
import { useDiscardGuard } from '../useDiscardGuard.tsx'
import { AppDialog, DialogFooter, type DialogSize } from './AppDialog.tsx'

const MULTILINE_ROWS = 3

/**
 * A dialog that edits something: Enter submits, Esc or Cancel with unsaved changes asks first, and while the form
 * cannot be submitted the footer says why instead of a button that stays mute.
 */
export function FormDialog({
  open,
  onClose,
  title,
  subtitle,
  icon,
  size = 'md',
  submitLabel,
  issue = '',
  dirty,
  busy = false,
  onSubmit,
  onBack,
  tabs,
  children,
}: {
  open: boolean
  onClose: () => void
  title: ReactNode
  subtitle?: ReactNode
  icon?: ReactNode
  size?: DialogSize
  submitLabel: string
  // The first thing that keeps the form from being submitted; empty when it can be.
  issue?: string
  dirty: boolean
  busy?: boolean
  onSubmit: () => void
  // A step back, for a form in steps: shown between Cancel and the submit button.
  onBack?: (() => void) | undefined
  tabs?: ReactNode
  children: ReactNode
}) {
  const { t } = useLingui()
  const guard = useDiscardGuard(dirty, onClose)
  const blocked = busy || issue !== ''
  const formId = useId()
  return (
    <AppDialog
      open={open}
      onClose={busy ? () => undefined : guard.request}
      title={title}
      subtitle={subtitle}
      icon={icon}
      size={size}
      tabs={tabs}
      footer={
        <DialogFooter
          aside={
            issue ? (
              <Box
                component="span"
                sx={{ display: 'inline-flex', alignItems: 'center', gap: 0.75 }}
              >
                <CircleAlert size={14} aria-hidden={true} />
                {issue}
              </Box>
            ) : null
          }
        >
          <Button type="button" onClick={guard.request} disabled={busy}>
            {t`Cancel`}
          </Button>
          {onBack ? (
            <Button type="button" onClick={onBack} disabled={busy}>
              {t`Back`}
            </Button>
          ) : null}
          <Button
            type="submit"
            form={formId}
            variant="contained"
            disabled={blocked}
            startIcon={busy ? <CircularProgress size={14} color="inherit" /> : null}
          >
            {submitLabel}
          </Button>
        </DialogFooter>
      }
    >
      <Box
        component="form"
        id={formId}
        noValidate={true}
        onSubmit={(e) => {
          e.preventDefault()
          if (!blocked) {
            onSubmit()
          }
        }}
        sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}
      >
        {children}
      </Box>
      {guard.dialog}
    </AppDialog>
  )
}

/** A group of related fields under a heading, set off from the group before it by a rule. */
export function FormSection({
  title,
  description,
  children,
}: {
  title: string
  description?: string
  children: ReactNode
}) {
  return (
    <Box
      component="section"
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: 2,
        '& + &': { borderTop: '1px solid var(--mortar-hairline-muted)', pt: 3 },
      }}
    >
      <Box>
        <Typography component="h3" sx={{ fontSize: 'lead.fontSize', fontWeight: 600 }}>
          {title}
        </Typography>
        {description ? (
          <Typography sx={{ fontSize: 'medium.fontSize', color: 'text.secondary', mt: 0.25 }}>
            {description}
          </Typography>
        ) : null}
      </Box>
      {children}
    </Box>
  )
}

/** Two short fields of equal width side by side; anything longer gets its own row. */
export function FieldRow({ children }: { children: ReactNode }) {
  return <Box sx={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 2 }}>{children}</Box>
}

/**
 * One field: its label above it, never inside the border; a hint under it that an error replaces; and a counter beside
 * the label when the value has a length limit.
 */
export function Field({
  label,
  hint,
  error,
  counter,
  children,
  htmlFor,
}: {
  label: string
  hint?: ReactNode | undefined
  error?: string | undefined
  counter?: string
  children: ReactNode
  htmlFor?: string
}) {
  return (
    <Box sx={{ minWidth: 0 }}>
      <Box
        component="label"
        htmlFor={htmlFor}
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          gap: 1,
          fontSize: 'medium.fontSize',
          fontWeight: 500,
          mb: 0.75,
        }}
      >
        <span>{label}</span>
        {counter ? (
          <Box component="span" sx={{ color: 'text.secondary', fontWeight: 400 }}>
            {counter}
          </Box>
        ) : null}
      </Box>
      {children}
      {error || hint ? (
        <Typography
          sx={{
            fontSize: 'small.fontSize',
            mt: 0.75,
            color: error ? 'error.main' : 'text.secondary',
          }}
        >
          {error || hint}
        </Typography>
      ) : null}
    </Box>
  )
}

/** A text field in a Field: label above, the value's limit counted beside the label. */
export function TextInput({
  label,
  value,
  onChange,
  hint,
  error,
  maxLength,
  multiline = false,
  placeholder,
  autoFocus = false,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  hint?: ReactNode
  error?: string
  maxLength?: number
  multiline?: boolean
  placeholder?: string
  autoFocus?: boolean
}) {
  const id = useId()
  return (
    <Field
      label={label}
      hint={hint}
      error={error}
      htmlFor={id}
      counter={maxLength ? `${value.length}/${maxLength}` : ''}
    >
      <TextField
        id={id}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        fullWidth={true}
        size="small"
        multiline={multiline}
        minRows={multiline ? MULTILINE_ROWS : undefined}
        placeholder={placeholder}
        autoFocus={autoFocus}
        error={Boolean(error)}
        slotProps={maxLength === undefined ? undefined : { htmlInput: { maxLength } }}
      />
    </Field>
  )
}

/** A dropdown in a Field: label above, never in the border. */
export function SelectField({
  label,
  value,
  onChange,
  hint,
  error,
  disabled = false,
  displayEmpty = false,
  children,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  hint?: ReactNode | undefined
  error?: string | undefined
  disabled?: boolean
  // Shows the empty value's item (a "Use default" choice) instead of a blank field.
  displayEmpty?: boolean
  children: ReactNode
}) {
  const id = useId()
  return (
    <Field label={label} htmlFor={id} hint={hint} error={error}>
      <TextField
        id={id}
        select={true}
        size="small"
        fullWidth={true}
        disabled={disabled}
        error={Boolean(error)}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        slotProps={{ select: { displayEmpty } }}
      >
        {children}
      </TextField>
    </Field>
  )
}

/** A value the form works out from the others, shown where it lands and not editable. */
export function DerivedValue({
  label,
  value,
  hint,
}: {
  label: string
  value: string
  hint?: ReactNode
}) {
  return (
    <Field label={label} hint={hint}>
      <Box
        sx={{
          px: 1.5,
          py: 1,
          borderRadius: '6px',
          bgcolor: 'var(--mortar-raised)',
          fontFamily: 'monospace',
          fontSize: 'medium.fontSize',
          overflowWrap: 'anywhere',
        }}
      >
        {value}
      </Box>
    </Field>
  )
}
