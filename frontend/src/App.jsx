import { BrowserRouter } from 'react-router-dom'
import GlassBackdrop from './components/layout/GlassBackdrop'
import AuthProvider from './context/AuthProvider'
import ThemeProvider from './context/ThemeProvider'
import ConfirmProvider from './components/ui/ConfirmProvider'
import ToastProvider from './components/ui/ToastProvider'
import Router from './router'
import AppBanner from './components/layout/AppBanner'
import UpdateBanner from './components/layout/UpdateBanner'
import { useKeyboardAwareFocus } from './lib/keyboard'

export default function App() {
  useKeyboardAwareFocus()
  return (
    <BrowserRouter>
      <ThemeProvider>
        <AuthProvider>
          <ToastProvider>
            <ConfirmProvider>
              <GlassBackdrop />
              <Router />
              <AppBanner />
              <UpdateBanner />
            </ConfirmProvider>
          </ToastProvider>
        </AuthProvider>
      </ThemeProvider>
    </BrowserRouter>
  )
}
