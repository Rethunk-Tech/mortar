import type { I18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { formatBytes } from '../i18n/bytes.ts'

interface LogReaders {
  live: () => Promise<string>
  run: (id: string) => Promise<string>
  runs: () => Promise<{ id: string }[] | null>
}

export function shareLogConfirm(i18n: I18n, bytes: number): string {
  const size = formatBytes(bytes)
  return i18n._(msg`This log is ${size}. Sharing uploads it to smapi.io as a public link.`)
}

export function pasteLogConfirm(i18n: I18n, bytes: number, site: string): string {
  const size = formatBytes(bytes)
  const { host } = new URL(site)
  return i18n._(
    msg`This log is ${size}. Sharing copies it and opens ${host}; paste it there to get a public link.`,
  )
}

// The live log Support reads is SMAPI's; a paste loader's log lives in each profile, so it shares the newest run.
export async function shareLogText(
  viewingRun: string,
  paste: boolean,
  read: LogReaders,
): Promise<string> {
  if (viewingRun) {
    return read.run(viewingRun)
  }
  if (!paste) {
    return read.live()
  }
  const newest = (await read.runs())?.[0]
  return newest ? read.run(newest.id) : ''
}
