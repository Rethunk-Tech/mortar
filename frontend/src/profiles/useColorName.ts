import { useLingui } from '@lingui/react/macro'

// useColorName names a profile or category colour token in the reader's language.
export function useColorName(): (token: string) => string {
  const { t } = useLingui()
  return (token) => {
    switch (token) {
      case 'rose':
        return t`Rose`
      case 'orange':
        return t`Orange`
      case 'gold':
        return t`Gold`
      case 'lime':
        return t`Lime`
      case 'teal':
        return t`Teal`
      case 'sky':
        return t`Sky`
      case 'violet':
        return t`Violet`
      case 'pink':
        return t`Pink`
      default:
        return token
    }
  }
}
