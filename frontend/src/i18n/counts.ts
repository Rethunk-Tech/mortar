import { plural } from '@lingui/core/macro'

const modsLabel = (count: number) => plural(count, { one: '# mod', other: '# mods' })
const updatesLabel = (count: number) => plural(count, { one: '# update', other: '# updates' })
const errorsLabel = (count: number) => plural(count, { one: '# error', other: '# errors' })
const problemsLabel = (count: number) => plural(count, { one: '# problem', other: '# problems' })

export { errorsLabel, modsLabel, problemsLabel, updatesLabel }
