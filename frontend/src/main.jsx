import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App'
import { hideSplashWhenReady } from './lib/splash'
import { bootstrapTheme } from './lib/theme'
import './index.css'

bootstrapTheme()

const root = document.getElementById('root')

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
)

hideSplashWhenReady(root)
