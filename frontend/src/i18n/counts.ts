import { plural } from '@lingui/core/macro'

const modsLabel = (count: number) => plural(count, { one: '# mod', other: '# mods' })
const updatesLabel = (count: number) => plural(count, { one: '# update', other: '# updates' })
const errorsLabel = (count: number) => plural(count, { one: '# error', other: '# errors' })

export { errorsLabel, modsLabel, updatesLabel }
