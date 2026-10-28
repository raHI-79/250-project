import { BrowserRouter } from 'react-router-dom'
import GlassBackdrop from './components/layout/GlassBackdrop'
import AuthProvider from './context/AuthProvider'
import ThemeProvider from './context/ThemeProvider'
import ConfirmProvider from './components/ui/ConfirmProvider'
import ToastProvider from './components/ui/ToastProvider'
import Router from './router'
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
            </ConfirmProvider>
          </ToastProvider>
        </AuthProvider>
      </ThemeProvider>
    </BrowserRouter>
  )
}
