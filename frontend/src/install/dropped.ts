const MORTAR_FILE = /\.mortar$/i

// A dropped .mortar file opens Import; everything else is an archive for the open profile.
export function splitDropped(paths: readonly string[]): { archives: string[]; mortar: string } {
  return {
    archives: paths.filter((p) => !MORTAR_FILE.test(p)),
    mortar: paths.find((p) => MORTAR_FILE.test(p)) ?? '',
  }
}
