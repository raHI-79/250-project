import { createContext, useContext } from 'react'

export const ChromeContext = createContext(null)

export const useChrome = () => useContext(ChromeContext)
