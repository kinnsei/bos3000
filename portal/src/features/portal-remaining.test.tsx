import { describe, it, expect, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/utils'

// ---------------------------------------------------------------------------
// Mock hooks – return deterministic data so assertions are stable
// ---------------------------------------------------------------------------
const mockAuthData = {
  id: 1,
  username: 'TestCorp',
  email: 'test@example.com',
  phone: '13800138000',
  api_key: 'sk-test1234567890abcdef12345678',
  ip_whitelist: ['10.0.0.1', '192.168.1.100'],
}

vi.mock('@/lib/api/hooks', () => ({
  useAuth: () => ({ data: mockAuthData, isLoading: false }),
  useUpdateProfile: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useChangePassword: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useRegenerateApiKey: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useMyDIDs: () => ({ data: ['010-12345678', '021-87654321'], isLoading: false }),
  useIpWhitelist: () => ({ data: ['10.0.0.1', '192.168.1.100'], isLoading: false }),
  useAddIp: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useRemoveIp: () => ({ mutateAsync: vi.fn(), isPending: false }),
}))

// ---------------------------------------------------------------------------
// Imports (after mocks)
// ---------------------------------------------------------------------------
import { ProfileForm } from '@/features/settings/profile-form'
import { SecuritySettings } from '@/features/settings/security-settings'
import { ApiKeys } from '@/features/api-integration/api-keys'
import { IpWhitelist } from '@/features/api-integration/ip-whitelist'

// ===========================
// Settings
// ===========================
describe('ProfileForm', () => {
  it('renders form fields (email, phone, company)', () => {
    renderWithProviders(<ProfileForm />)

    expect(screen.getByLabelText('邮箱')).toBeInTheDocument()
    expect(screen.getByLabelText('手机号码')).toBeInTheDocument()
    expect(screen.getByLabelText('公司名称')).toBeInTheDocument()
  })

  it('shows company name as read-only', () => {
    renderWithProviders(<ProfileForm />)

    const companyInput = screen.getByLabelText('公司名称')
    expect(companyInput).toHaveAttribute('readOnly')
  })
})

describe('SecuritySettings', () => {
  it('renders password fields', () => {
    renderWithProviders(<SecuritySettings />)

    expect(screen.getByLabelText('当前密码')).toBeInTheDocument()
    expect(screen.getByLabelText('新密码')).toBeInTheDocument()
    expect(screen.getByLabelText('确认新密码')).toBeInTheDocument()
  })

  it('has a submit button to change password', () => {
    renderWithProviders(<SecuritySettings />)

    expect(screen.getByRole('button', { name: '修改密码' })).toBeInTheDocument()
  })
})

// ===========================
// API Integration
// ===========================
describe('ApiKeys', () => {
  it('shows masked API key', () => {
    renderWithProviders(<ApiKeys />)

    expect(screen.getByText('sk-test1************************')).toBeInTheDocument()
  })

  it('has regenerate button', () => {
    renderWithProviders(<ApiKeys />)

    const buttons = screen.getAllByRole('button', { name: /重置密钥/ })
    expect(buttons.length).toBeGreaterThan(0)
  })
})

describe('IpWhitelist', () => {
  it('renders IP list from user data', () => {
    renderWithProviders(<IpWhitelist />)

    expect(screen.getByText('10.0.0.1')).toBeInTheDocument()
    expect(screen.getByText('192.168.1.100')).toBeInTheDocument()
  })

  it('has add IP input and button', () => {
    renderWithProviders(<IpWhitelist />)

    expect(screen.getByPlaceholderText(/输入 IP 地址/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /添加/ })).toBeInTheDocument()
  })
})
