import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Button } from '@/components/ui/button'
import { useLogin } from '@/lib/api/hooks'
import { getErrorMessage } from '@/lib/api/error'
import { Logo, LogoIcon } from '@/components/shared/logo'
import { Eye, EyeOff, Shield, Lock, Mail } from 'lucide-react'

export function LoginPage() {
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [error, setError] = useState('')
  const [attempts, setAttempts] = useState(0)
  const navigate = useNavigate()
  const login = useLogin()

  const isLocked = attempts >= 5

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    if (isLocked) {
      setError('登录尝试过多，请稍后再试')
      return
    }
    setError('')
    try {
      await login.mutateAsync({ email, password })
      navigate('/', { replace: true })
    } catch (err) {
      setAttempts((a) => a + 1)
      setError(getErrorMessage(err))
    }
  }

  return (
    <div className="flex min-h-screen">
      {/* Left branding panel */}
      <div className="hidden lg:flex lg:w-1/2 relative overflow-hidden bg-gradient-to-br from-primary/90 via-primary to-primary/80">
        {/* Animated background pattern */}
        <div className="absolute inset-0 opacity-10">
          <div className="absolute top-1/4 left-1/4 h-64 w-64 rounded-full bg-white/20 blur-3xl animate-pulse" />
          <div className="absolute bottom-1/4 right-1/4 h-96 w-96 rounded-full bg-white/10 blur-3xl animate-pulse [animation-delay:1s]" />
          <div className="absolute top-1/2 left-1/2 h-48 w-48 rounded-full bg-white/15 blur-3xl animate-pulse [animation-delay:2s]" />
        </div>
        {/* Grid pattern overlay */}
        <div className="absolute inset-0" style={{
          backgroundImage: 'radial-gradient(circle at 1px 1px, rgba(255,255,255,0.08) 1px, transparent 0)',
          backgroundSize: '40px 40px',
        }} />
        {/* Content */}
        <div className="relative z-10 flex flex-col justify-center px-16 text-white">
          <LogoIcon size="xl" className="text-white mb-8" />
          <h1 className="text-4xl font-bold mb-4">BOS3000</h1>
          <p className="text-xl text-white/80 mb-2">Business Operations System</p>
          <p className="text-white/60 max-w-md leading-relaxed">
            企业级 VoIP 回拨运营平台。实时监控、智能路由、全链路话单追踪。
          </p>
          <div className="mt-12 flex items-center gap-8 text-white/60 text-sm">
            <div className="flex items-center gap-2">
              <Shield className="h-4 w-4" />
              <span>安全加密</span>
            </div>
            <div className="flex items-center gap-2">
              <Lock className="h-4 w-4" />
              <span>权限管控</span>
            </div>
          </div>
        </div>
      </div>

      {/* Right login form */}
      <div className="flex w-full lg:w-1/2 items-center justify-center bg-background px-6">
        <div className="w-full max-w-sm space-y-8">
          {/* Mobile logo */}
          <div className="lg:hidden flex justify-center">
            <Logo size="lg" />
          </div>

          <div>
            <h2 className="text-2xl font-bold tracking-tight">管理后台</h2>
            <p className="text-muted-foreground mt-1">请使用管理员账号登录</p>
          </div>

          <form onSubmit={handleSubmit} className="space-y-5">
            <div className="space-y-2">
              <Label htmlFor="email">邮箱地址</Label>
              <div className="relative">
                <Mail className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  id="email"
                  type="email"
                  placeholder="admin@example.com"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  className="h-10 pl-10"
                  autoComplete="email"
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="password">密码</Label>
              <div className="relative">
                <Lock className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  className="h-10 pl-10 pr-10"
                  autoComplete="current-password"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                  tabIndex={-1}
                >
                  {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                </button>
              </div>
            </div>

            {error && (
              <div className="rounded-md bg-destructive/10 border border-destructive/20 px-3 py-2">
                <p className="text-sm text-destructive">{error}</p>
              </div>
            )}

            {isLocked && (
              <div className="rounded-md bg-amber-500/10 border border-amber-500/20 px-3 py-2">
                <p className="text-sm text-amber-600 dark:text-amber-400">
                  登录尝试次数过多，请等待 30 秒后重试
                </p>
              </div>
            )}

            <Button
              type="submit"
              className="w-full h-10 font-medium"
              disabled={login.isPending || isLocked}
            >
              {login.isPending ? (
                <span className="flex items-center gap-2">
                  <span className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
                  登录中...
                </span>
              ) : (
                '登录'
              )}
            </Button>
          </form>

          <p className="text-center text-xs text-muted-foreground">
            BOS3000 &copy; {new Date().getFullYear()} &middot; 安全连接
          </p>
        </div>
      </div>
    </div>
  )
}
