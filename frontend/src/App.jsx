import { BrowserRouter } from 'react-router-dom'
import GlassBackdrop from './components/layout/GlassBackdrop'
import AuthProvider from './context/AuthProvider'
import ThemeProvider from './context/ThemeProvider'
import ConfirmProvider from './components/ui/ConfirmProvider'
import ConfirmPasswordProvider from './components/ui/ConfirmPasswordProvider'
import ToastProvider from './components/ui/ToastProvider'
import Router from './router'
import ErrorBoundary from './components/ui/ErrorBoundary'
import { useNativeBackButton } from './lib/shell'
import SiteGate from './components/layout/SiteGate'
import AppBanner from './components/layout/AppBanner'
import AppUpdate from './components/layout/AppUpdate'
import UpdateBanner from './components/layout/UpdateBanner'
import { useKeyboardAwareFocus } from './lib/keyboard'

function BackButton() {
  useNativeBackButton()
  return null
}

export default function App() {
  useKeyboardAwareFocus()
  return (
    <BrowserRouter>
      <ThemeProvider>
        <AuthProvider>
          <ToastProvider>
            <ConfirmProvider>
              <ConfirmPasswordProvider>
              <GlassBackdrop />
              <BackButton />
              <ErrorBoundary>
                <SiteGate>
                  <Router />
                </SiteGate>
              </ErrorBoundary>
              <AppBanner />
              <AppUpdate />
              <UpdateBanner />
              </ConfirmPasswordProvider>
            </ConfirmProvider>
          </ToastProvider>
        </AuthProvider>
      </ThemeProvider>
    </BrowserRouter>
  )
}
