import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'

export interface PrefCopy {
  label: string
  description?: string
  placeholder?: string
  options?: { value: string; label: string; hint?: string }[]
}

export function antivirusPrefs(i18n: I18n): Record<string, PrefCopy> {
  return {
    antivirus: {
      label: i18n._(msg`Antivirus`),
      description: i18n._(
        msg`Scan every mod after it is extracted and before it enters Mortar's store. This applies to every game.`,
      ),
      options: [
        {
          value: 'automatic',
          label: i18n._(msg`Automatic`),
          hint: i18n._(msg`Windows' antivirus through AMSI; clamd on Linux when it is running.`),
        },
        {
          value: 'clamd',
          label: i18n._(msg`clamd socket`),
          hint: i18n._(msg`Ask a ClamAV daemon at the socket path below.`),
        },
        {
          value: 'command',
          label: i18n._(msg`Custom command`),
          hint: i18n._(msg`Run your own scanner with the command below.`),
        },
        {
          value: 'off',
          label: i18n._(msg`Off`),
          hint: i18n._(msg`No scanning. The mod sites' own scans still apply.`),
        },
      ],
    },
    antivirusSocket: {
      label: i18n._(msg`clamd socket path`),
      description: i18n._(
        msg`Used with clamd. Empty tries the usual places, such as /run/clamav/clamd.ctl.`,
      ),
    },
    antivirusCommand: {
      label: i18n._(msg`Scan command`),
      description: i18n._(
        msg`Used with Custom command. {path} stands for the folder to scan; exit code 0 means clean, any other code a detection, and the first line it prints names it.`,
      ),
    },
  }
}
