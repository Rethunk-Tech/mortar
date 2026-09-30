import { useTheme } from '@mui/material/styles'

export function Logo({ size = 20, fill }: { size?: number; fill?: string }) {
  const accent = useTheme().palette.primary.main
  return (
    <svg width={size} height={size} viewBox="0 0 64 64" aria-hidden="true" fill={fill ?? accent}>
      <rect x="6" y="12" width="24" height="11" rx="2" />
      <rect x="34" y="12" width="24" height="11" rx="2" />
      <rect x="6" y="27" width="10" height="11" rx="2" />
      <rect x="20" y="27" width="24" height="11" rx="2" />
      <rect x="48" y="27" width="10" height="11" rx="2" />
      <rect x="6" y="42" width="24" height="11" rx="2" />
      <rect x="34" y="42" width="24" height="11" rx="2" />
    </svg>
  )
}
