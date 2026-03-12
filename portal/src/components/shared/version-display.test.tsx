import { describe, it, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/utils'
import { APP_VERSION } from '@/lib/version'

// Mock heavy dependencies used by Layout
vi.mock('@/lib/api/hooks', () => ({
  useAuth: () => ({
    data: { username: 'testuser' },
    isLoading: false,
    isError: false,
  }),
  useLogin: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
}))

vi.mock('@/hooks/use-call-ws', () => ({
  useCallWs: () => ({ status: 'disconnected', events: [] }),
}))

describe('Version display', () => {
  it('APP_VERSION is defined and follows semver format', () => {
    expect(APP_VERSION).toBeDefined()
    expect(APP_VERSION).toMatch(/^\d+\.\d+\.\d+$/)
  })

  it('APP_VERSION is 0.8.0', () => {
    expect(APP_VERSION).toBe('0.8.0')
  })

  it('renders version string in the portal layout header', async () => {
    const { Layout } = await import('@/app/layout')

    renderWithProviders(<Layout />)

    const versionEl = screen.getByText(`v${APP_VERSION}`)
    expect(versionEl).toBeInTheDocument()
    expect(versionEl).toHaveClass('font-mono')
  })
})
