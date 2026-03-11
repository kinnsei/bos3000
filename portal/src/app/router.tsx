import { createBrowserRouter, RouterProvider } from 'react-router'
import { lazy, Suspense } from 'react'
import { Layout } from './layout'
import { LoginPage } from '@/features/auth/login'
import { SkeletonPage } from '@/components/shared/skeleton-page'

const Dashboard = lazy(() => import('@/features/dashboard'))
const Callback = lazy(() => import('@/features/callback'))
const CDR = lazy(() => import('@/features/cdr'))
const Wastage = lazy(() => import('@/features/wastage'))
const Finance = lazy(() => import('@/features/finance'))
const ApiIntegration = lazy(() => import('@/features/api-integration'))
const PortalSettings = lazy(() => import('@/features/settings'))

function LazyPage({ Component }: { Component: React.LazyExoticComponent<() => React.JSX.Element> }) {
  return (
    <Suspense fallback={<SkeletonPage />}>
      <Component />
    </Suspense>
  )
}

const router = createBrowserRouter([
  {
    path: '/login',
    element: <LoginPage />,
  },
  {
    path: '/',
    element: <Layout />,
    children: [
      { index: true, element: <LazyPage Component={Dashboard} /> },
      { path: 'callback', element: <LazyPage Component={Callback} /> },
      { path: 'cdr', element: <LazyPage Component={CDR} /> },
      { path: 'wastage', element: <LazyPage Component={Wastage} /> },
      { path: 'finance', element: <LazyPage Component={Finance} /> },
      { path: 'api-integration', element: <LazyPage Component={ApiIntegration} /> },
      { path: 'settings', element: <LazyPage Component={PortalSettings} /> },
    ],
  },
])

export function AppRouter() {
  return <RouterProvider router={router} />
}
