import { GlobalRegistrator } from '@happy-dom/global-registrator'

// One DOM for the whole run: a file that registered and unregistered its own left the testing library's cached
// `document` pointing at a closed window for every component test after it.
GlobalRegistrator.register({ url: 'http://localhost/' })
Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true })
