import { readFileSync } from 'node:fs'
import { transformSync } from '@babel/core'
import linguiMacro from '@lingui/babel-plugin-lingui-macro'
import { getConfig } from '@lingui/conf'

const linguiConfig = getConfig({ cwd: new URL('..', import.meta.url).pathname })

// Vite compiles Lingui's macros in the app; bun test does not, so component tests get the same transform here.
Bun.plugin({
  name: 'lingui-macro',
  setup(build) {
    build.onLoad({ filter: /frontend[\\/]src[\\/].*\.tsx?$/ }, ({ path }) => {
      const source = readFileSync(path, 'utf8')
      const loader = path.endsWith('x') ? 'tsx' : 'ts'
      if (!source.includes('@lingui/')) {
        return { contents: source, loader }
      }
      const out = transformSync(source, {
        filename: path,
        babelrc: false,
        configFile: false,
        parserOpts: { plugins: ['typescript', 'jsx'] },
        plugins: [[linguiMacro, { linguiConfig }]],
      })
      return { contents: out?.code ?? source, loader }
    })
  },
})
