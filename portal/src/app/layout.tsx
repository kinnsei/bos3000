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
import { Logo, LogoIcon } from '@/components/shared/logo'
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
        <Sidebar collapsible="icon">
          <SidebarHeader className="border-b px-4 py-3 group-data-[collapsible=icon]:px-0 group-data-[collapsible=icon]:py-3">
            {/* Expanded: full logo with text */}
            <div className="group-data-[collapsible=icon]:hidden">
              <Logo size="sm" />
              <span className="text-xs text-muted-foreground mt-0.5 block">客户门户</span>
            </div>
            {/* Collapsed: icon only, centered */}
            <div className="hidden group-data-[collapsible=icon]:flex group-data-[collapsible=icon]:justify-center">
              <LogoIcon size="sm" />
            </div>
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
          <SidebarFooter className="border-t p-4 group-data-[collapsible=icon]:p-2">
            <div className="flex items-center gap-2 text-sm text-muted-foreground group-data-[collapsible=icon]:justify-center">
              <Avatar className="h-6 w-6">
                <AvatarFallback className="text-xs">
                  {user?.username?.[0]?.toUpperCase() ?? 'U'}
                </AvatarFallback>
              </Avatar>
              <span className="truncate group-data-[collapsible=icon]:hidden">{user?.username ?? '用户'}</span>
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
            <DropdownMenu>
              <DropdownMenuTrigger className="inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium ring-offset-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50 hover:bg-accent hover:text-accent-foreground h-9 w-9 cursor-pointer">
                  <Avatar className="h-7 w-7">
                    <AvatarFallback className="text-xs">
                      {user?.username?.[0]?.toUpperCase() ?? 'U'}
                    </AvatarFallback>
                  </Avatar>
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
          <WsStatusBanner status={wsStatus} />
          <main className="flex-1 p-6">
            <Outlet />
          </main>
        </SidebarInset>
      </SidebarProvider>
    </AuthGuard>
  )
}
