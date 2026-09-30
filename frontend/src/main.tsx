import '@wailsio/runtime'
import { CssBaseline, ThemeProvider } from '@mui/material'
import React from 'react'
import ReactDOM from 'react-dom/client'
import { App } from './App.tsx'
import { defaultAccent } from './theme/accents.ts'
import './theme/fonts.ts'
import { createMortarTheme } from './theme/theme.ts'

const theme = createMortarTheme(defaultAccent)

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <App />
    </ThemeProvider>
  </React.StrictMode>,
)
