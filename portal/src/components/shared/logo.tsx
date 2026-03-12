import { cn } from '@/lib/utils'

interface LogoProps {
  size?: 'sm' | 'md' | 'lg' | 'xl'
  showText?: boolean
  className?: string
}

const sizes = {
  sm: 'h-6 w-6',
  md: 'h-8 w-8',
  lg: 'h-10 w-10',
  xl: 'h-16 w-16',
}

const textSizes = {
  sm: 'text-sm',
  md: 'text-lg',
  lg: 'text-xl',
  xl: 'text-3xl',
}

export function LogoIcon({ size = 'md', className }: { size?: LogoProps['size']; className?: string }) {
  return (
    <svg
      viewBox="0 0 48 48"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={cn(sizes[size], className)}
    >
      {/* Outer hexagon frame */}
      <path
        d="M24 2L43.6 13v22L24 46 4.4 35V13L24 2z"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        className="text-primary"
        strokeLinejoin="round"
      />
      {/* Inner signal waves - 3 arcs radiating from center-left */}
      <path
        d="M18 30a8 8 0 0 1 0-12"
        fill="none"
        stroke="currentColor"
        strokeWidth="2.5"
        strokeLinecap="round"
        className="text-primary"
      />
      <path
        d="M22 28a5 5 0 0 1 0-8"
        fill="none"
        stroke="currentColor"
        strokeWidth="2.5"
        strokeLinecap="round"
        className="text-primary"
      />
      <path
        d="M26 26a2 2 0 0 1 0-4"
        fill="none"
        stroke="currentColor"
        strokeWidth="2.5"
        strokeLinecap="round"
        className="text-primary"
      />
      {/* Connection node dot */}
      <circle cx="30" cy="24" r="3" fill="currentColor" className="text-primary" />
      {/* Two small satellite dots */}
      <circle cx="36" cy="18" r="1.5" fill="currentColor" className="text-primary/60" />
      <circle cx="36" cy="30" r="1.5" fill="currentColor" className="text-primary/60" />
      {/* Lines from main node to satellites */}
      <line x1="30" y1="24" x2="36" y2="18" stroke="currentColor" strokeWidth="1" className="text-primary/40" />
      <line x1="30" y1="24" x2="36" y2="30" stroke="currentColor" strokeWidth="1" className="text-primary/40" />
    </svg>
  )
}

export function Logo({ size = 'md', showText = true, className }: LogoProps) {
  return (
    <div className={cn('flex items-center gap-2', className)}>
      <LogoIcon size={size} />
      {showText && (
        <span className={cn('font-bold tracking-tight', textSizes[size])}>
          BOS<span className="text-primary">3000</span>
        </span>
      )}
    </div>
  )
}
