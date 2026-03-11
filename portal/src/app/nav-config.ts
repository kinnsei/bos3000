import {
  LayoutDashboard,
  PhoneCall,
  FileText,
  TrendingDown,
  Wallet,
  Code2,
  Settings,
} from 'lucide-react'

export const menuGroups = [
  {
    label: '概览',
    items: [
      { label: '仪表盘', icon: LayoutDashboard, path: '/' },
    ],
  },
  {
    label: '业务',
    items: [
      { label: '回拨操作', icon: PhoneCall, path: '/callback' },
      { label: '话单查询', icon: FileText, path: '/cdr' },
      { label: '损耗分析', icon: TrendingDown, path: '/wastage' },
    ],
  },
  {
    label: '账务',
    items: [
      { label: '财务中心', icon: Wallet, path: '/finance' },
    ],
  },
  {
    label: '开发',
    items: [
      { label: 'API集成', icon: Code2, path: '/api-integration' },
    ],
  },
  {
    label: '设置',
    items: [
      { label: '账户设置', icon: Settings, path: '/settings' },
    ],
  },
]
