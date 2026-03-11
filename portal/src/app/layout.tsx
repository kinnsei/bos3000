import { Outlet, useNavigate, useLocation } from 'react-router'
import {
  Sun,
  Moon,
  LogOut,
  Settings,
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
import { menuGroups } from './nav-config'
import { useEffect } from 'react'
import { APP_VERSION } from '@/lib/version'

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
            <span className="text-xs text-muted-foreground">客户门户</span>
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
                  {user?.username?.[0]?.toUpperCase() ?? 'U'}
                </AvatarFallback>
              </Avatar>
              <span className="truncate">{user?.username ?? '用户'}</span>
            </div>
          </SidebarFooter>
        </Sidebar>
        <SidebarInset>
          <WsStatusBanner status={wsStatus} />
          <header className="flex h-14 items-center gap-2 border-b px-4">
            <SidebarTrigger />
            <Separator orientation="vertical" className="h-6" />
            <div className="flex-1" />
            <span className="text-xs text-muted-foreground font-mono">v{APP_VERSION}</span>
            <Button variant="ghost" size="icon" onClick={toggleTheme}>
              {resolvedTheme === 'dark' ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
            </Button>
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon">
                  <Avatar className="h-7 w-7">
                    <AvatarFallback className="text-xs">
                      {user?.username?.[0]?.toUpperCase() ?? 'U'}
                    </AvatarFallback>
                  </Avatar>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => navigate('/settings')}>
                  <Settings className="mr-2 h-4 w-4" />
                  账户设置
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => navigate('/login')}>
                  <LogOut className="mr-2 h-4 w-4" />
                  退出登录
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </header>
          <main className="flex-1 p-6">
            <Outlet />
          </main>
        </SidebarInset>
      </SidebarProvider>
    </AuthGuard>
  )
}
