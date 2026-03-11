import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import '../index.css'
import { Providers } from './providers'
import { AppRouter } from './router'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <Providers>
      <AppRouter />
    </Providers>
  </StrictMode>,
)
