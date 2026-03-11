import { createBrowserRouter, RouterProvider } from 'react-router'
import { lazy, Suspense } from 'react'
import { Layout } from './layout'
import { LoginPage } from '@/features/auth/login'
import { SkeletonPage } from '@/components/shared/skeleton-page'

const Dashboard = lazy(() => import('@/features/dashboard'))
const Customers = lazy(() => import('@/features/customers'))
const Gateways = lazy(() => import('@/features/gateways'))
const CDR = lazy(() => import('@/features/cdr'))
const Wastage = lazy(() => import('@/features/wastage'))
const Finance = lazy(() => import('@/features/finance'))
const DID = lazy(() => import('@/features/did'))
const Compliance = lazy(() => import('@/features/compliance'))
const Ops = lazy(() => import('@/features/ops'))
const Settings = lazy(() => import('@/features/settings'))

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
      { path: 'customers', element: <LazyPage Component={Customers} /> },
      { path: 'customers/:id', element: <LazyPage Component={Customers} /> },
      { path: 'gateways', element: <LazyPage Component={Gateways} /> },
      { path: 'cdr', element: <LazyPage Component={CDR} /> },
      { path: 'wastage', element: <LazyPage Component={Wastage} /> },
      { path: 'finance', element: <LazyPage Component={Finance} /> },
      { path: 'did', element: <LazyPage Component={DID} /> },
      { path: 'compliance', element: <LazyPage Component={Compliance} /> },
      { path: 'ops', element: <LazyPage Component={Ops} /> },
      { path: 'settings', element: <LazyPage Component={Settings} /> },
    ],
  },
])

export function AppRouter() {
  return <RouterProvider router={router} />
}
