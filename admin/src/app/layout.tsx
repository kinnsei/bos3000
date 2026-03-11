import { Outlet, useNavigate, useLocation } from 'react-router'
import {
  LayoutDashboard,
  PhoneCall,
  TrendingDown,
  Users,
  Network,
  Hash,
  Wallet,
  Shield,
  Activity,
  Settings,
  Sun,
  Moon,
  Bell,
  LogOut,
} from 'lucide-react'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarTrigger,
} from '@/components/ui/sidebar'
import { Button } from '@/components/ui/button'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Separator } from '@/components/ui/separator'
import { useTheme } from './providers'
import { useAuth } from '@/lib/api/hooks'
import { SkeletonPage } from '@/components/shared/skeleton-page'
import { WsStatusBanner } from '@/components/shared/ws-status-banner'
import { useCallWs } from '@/hooks/use-call-ws'
import { useEffect } from 'react'
import { APP_VERSION } from '@/lib/version'

const menuGroups = [
  {
    label: '监控',
    items: [
      { label: '运营概览', icon: LayoutDashboard, path: '/' },
      { label: '话单管理', icon: PhoneCall, path: '/cdr' },
      { label: '损耗分析', icon: TrendingDown, path: '/wastage' },
    ],
  },
  {
    label: '管理',
    items: [
      { label: '客户管理', icon: Users, path: '/customers' },
      { label: '网关管理', icon: Network, path: '/gateways' },
      { label: 'DID 管理', icon: Hash, path: '/did' },
    ],
  },
  {
    label: '财务',
    items: [
      { label: '财务中心', icon: Wallet, path: '/finance' },
    ],
  },
  {
    label: '系统',
    items: [
      { label: '合规审计', icon: Shield, path: '/compliance' },
      { label: '运维监控', icon: Activity, path: '/ops' },
      { label: '系统设置', icon: Settings, path: '/settings' },
    ],
  },
]

function AuthGuard({ children }: { children: React.ReactNode }) {
  const { isLoading, isError } = useAuth()
  const navigate = useNavigate()

  useEffect(() => {
    if (isError) {
      navigate('/login', { replace: true })
    }
  }, [isError, navigate])

  if (isLoading) return <SkeletonPage />
  if (isError) return null

  return <>{children}</>
}

export function Layout() {
  const { resolvedTheme, setTheme } = useTheme()
  const location = useLocation()
  const navigate = useNavigate()
  const { data: user } = useAuth()
  const { status: wsStatus } = useCallWs()

  const toggleTheme = () => {
    setTheme(resolvedTheme === 'dark' ? 'light' : 'dark')
  }

  return (
    <AuthGuard>
      <SidebarProvider>
        <Sidebar>
          <SidebarHeader className="border-b px-4 py-3">
            <span className="text-lg font-bold tracking-tight">BOS3000</span>
          </SidebarHeader>
          <SidebarContent>
            {menuGroups.map((group) => (
              <SidebarGroup key={group.label}>
                <SidebarGroupLabel>{group.label}</SidebarGroupLabel>
                <SidebarGroupContent>
                  <SidebarMenu>
                    {group.items.map((item) => (
                      <SidebarMenuItem key={item.path}>
                        <SidebarMenuButton
                          isActive={location.pathname === item.path}
                          onClick={() => navigate(item.path)}
                          tooltip={item.label}
                        >
                          <item.icon />
                          <span>{item.label}</span>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    ))}
                  </SidebarMenu>
                </SidebarGroupContent>
              </SidebarGroup>
            ))}
          </SidebarContent>
          <SidebarFooter className="border-t p-4">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Avatar className="h-6 w-6">
                <AvatarFallback className="text-xs">
                  {user?.username?.[0]?.toUpperCase() ?? 'A'}
                </AvatarFallback>
              </Avatar>
              <span className="truncate">{user?.username ?? '管理员'}</span>
            </div>
          </SidebarFooter>
        </Sidebar>
        <SidebarInset>
          <header className="flex h-14 items-center gap-2 border-b px-4">
            <SidebarTrigger />
            <Separator orientation="vertical" className="h-6" />
            <div className="flex-1" />
            <span className="text-xs text-muted-foreground font-mono">v{APP_VERSION}</span>
            <Button variant="ghost" size="icon" onClick={toggleTheme}>
              {resolvedTheme === 'dark' ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
            </Button>
            <Button variant="ghost" size="icon">
              <Bell className="h-4 w-4" />
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon">
                  <Avatar className="h-7 w-7">
                    <AvatarFallback className="text-xs">
                      {user?.username?.[0]?.toUpperCase() ?? 'A'}
                    </AvatarFallback>
                  </Avatar>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => navigate('/login')}>
                  <LogOut className="mr-2 h-4 w-4" />
                  退出登录
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </header>
          <WsStatusBanner status={wsStatus} />
          <main className="flex-1 p-6">
            <Outlet />
          </main>
        </SidebarInset>
      </SidebarProvider>
    </AuthGuard>
  )
}
