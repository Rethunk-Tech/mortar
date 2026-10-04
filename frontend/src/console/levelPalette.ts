import { alpha, type Theme } from '@mui/material/styles'
import { Level } from '../../bindings/github.com/Rethunk-AI/mortar/internal/launch/models.ts'

const ROW_WARN = 0.08
const ROW_ERROR = 0.1
const ROW_ALERT = 0.1
const TRACE_ALPHA = 0.9
const DEBUG_ALPHA = 0.95

interface LevelChrome {
  color: string
  bar: string
  row: string
  text: string
}

const CLEAR: LevelChrome = { color: '', bar: 'transparent', row: 'transparent', text: '' }

function levelChrome(th: Theme, level: Level): LevelChrome {
  const { warning, error, info, text } = th.palette
  switch (level) {
    case Level.Warn:
      return {
        color: warning.main,
        bar: warning.main,
        row: alpha(warning.main, ROW_WARN),
        text: warning.light,
      }
    case Level.Error:
      return {
        color: error.light,
        bar: error.main,
        row: alpha(error.main, ROW_ERROR),
        text: error.light,
      }
    case Level.Alert:
      return {
        color: info.light,
        bar: info.main,
        row: alpha(info.main, ROW_ALERT),
        text: info.light,
      }
    case Level.Info:
      return {
        color: text.primary,
        bar: 'transparent',
        row: 'transparent',
        text: text.primary,
      }
    case Level.Debug:
      return {
        color: alpha(text.secondary, DEBUG_ALPHA),
        bar: 'transparent',
        row: 'transparent',
        text: alpha(text.secondary, DEBUG_ALPHA),
      }
    case Level.Trace:
      return {
        color: alpha(text.secondary, TRACE_ALPHA),
        bar: 'transparent',
        row: 'transparent',
        text: alpha(text.secondary, TRACE_ALPHA),
      }
    default:
      return CLEAR
  }
}

function levelSwatch(th: Theme, level: Level): string {
  return levelChrome(th, level).color
}

export { levelChrome, levelSwatch }
