# Phase 5: Client Portal - Research

**Researched:** 2026-03-10
**Domain:** React SPA (Client Portal) + Go WebSocket backend (Encore raw endpoint), mirroring Phase 4 Admin Dashboard stack
**Confidence:** HIGH

## Summary

Phase 5 builds the Client Portal as a standalone React SPA in `/portal`, structurally cloned from the Phase 4 Admin Dashboard (`/admin`). The frontend stack is identical: React 18 + Vite + TypeScript + Tailwind CSS v4 + shadcn/ui + TanStack Query v5 + React Router v7 (library mode) + Recharts. The key technical additions beyond Phase 4 are: (1) a WebSocket backend in the callback service using Encore raw endpoints + gorilla/websocket for real-time call status push, (2) a frontend WebSocket integration layer that updates TanStack Query cache on incoming events, (3) Excel/CSV export for CDR data using SheetJS, and (4) an HTML5 audio player for recording playback from presigned S3 URLs.

The Portal mirrors the Admin layout (collapsible sidebar + topnav) but with customer-scoped navigation (overview, callback, CDR, wastage, finance, API integration, settings). Authentication uses JWT Token (Bearer header) instead of Admin's JWT Cookie. The WebSocket endpoint lives in the callback service as an Encore raw endpoint, with per-user connection isolation via JWT validation on connect. The same WebSocket endpoint serves both Admin (all calls) and Portal (user-scoped calls), differentiated by the authenticated user's role.

**Primary recommendation:** Clone the Admin scaffold (Vite/shadcn/Tailwind config) into `/portal`, reuse all shared component patterns (DataTable, StatCard, chart wrapper), add a `useCallStatusWebSocket` custom hook that listens for call state events and invalidates/updates TanStack Query cache. Build the WebSocket Go backend using gorilla/websocket inside an Encore raw endpoint with a Hub pattern for per-user connection management. Admin Dashboard gets WebSocket upgrade in the same pass (replacing 30s polling for live call data).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Portal is a separate project in `/portal` directory (parallel to `/admin`), copying Admin scaffold configuration
- Theme and visual style identical to Admin: Vercel Dashboard style, green/cyan accent, dark/light theme
- Login page reuses Admin structure, title changed to "Customer Portal" ("客户门户")
- API client: `encore gen client --lang=typescript` generated to each project's `src/lib/client`
- Customer auth: email + password login, backend returns JWT Token, frontend stores in localStorage
- WebSocket only pushes call state change events (initiating -> a_dialing -> a_connected -> b_dialing -> bridged -> finished/failed)
- WebSocket endpoint in callback service (Encore raw endpoint), state machine changes push directly, no cross-service communication
- Per-user isolation: JWT validated on WS connect, server maintains connection pool by user_id; customers only see own calls, admins see all
- Admin Dashboard upgraded to WebSocket simultaneously (replacing Phase 4 30s polling)
- Exponential backoff reconnect on disconnect, top banner "real-time connection lost" indicator
- Other data (overview metrics, finance, CDR lists) continues using TanStack Query with timed refetch
- Batch import (Excel) REMOVED from CLNT-02 -- B2B customers use API for bulk, Web Portal for single operations only
- Callback page is an "operations center": initiate form (top) + recent callback history (bottom), with live call cards updated via WebSocket
- Active calls displayed as card list: each card shows A/B numbers, current status (colored badge), duration, hangup button
- Navigation: left collapsible sidebar grouped by customer functions: Overview, Callback, CDR, Wastage Analysis, Finance Center, API Integration, Account Settings
- Dashboard overview: top row 4-6 metric cards (today's calls, success rate, wastage rate, spend, balance, concurrent), 1-2 trend charts below
- CDR detail uses Sheet side drawer: A/B leg info sections, status timeline, recording player, cost breakdown
- Desktop-first design, tablet usable, no mobile optimization
- Chinese-primary UI, v1 no i18n

### Claude's Discretion
- shadcn/ui component selection and composition
- Callback form advanced options expand/collapse interaction details
- Active call card design and animation effects
- Dashboard chart configuration (Recharts)
- Recording player implementation in side drawer
- WebSocket reconnect specific backoff parameters
- CDR export format handling (Excel/CSV)
- Account settings page layout
- Frontend directory structure and file organization

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| CLNT-01 | Dashboard: today's overview (calls/success rate/wastage rate/spend/balance), concurrent calls | TanStack Query with refetchInterval for metrics; Recharts AreaChart for trends; StatCard component reused from Admin pattern |
| CLNT-02 | Callback ops: Web form to initiate callback (batch import removed), active call list with self-hangup | Callback form with shadcn Form + Zod validation; call cards with WebSocket live updates; useMutation for initiate/hangup |
| CLNT-03 | CDR: query/export (Excel/CSV), detail page (A/B leg split, hangup_cause, recording playback) | DataTable with server-side pagination; SheetJS for Excel export; Sheet side drawer with HTML5 audio player for recording |
| CLNT-04 | Finance: balance & credit, transaction history, rate lookup | StatCard for balance display; DataTable for transactions; rate query form with shadcn Select |
| CLNT-05 | Wastage analysis: overview trend, detail list, B-leg failure reason distribution | Recharts LineChart (trend), PieChart (failure distribution); DataTable for detail list; reuse chart-theme from Admin |
| CLNT-06 | API integration: API Key management, Webhook config/test/log, IP whitelist | Form-based CRUD pages; shadcn Input/Button for key management; webhook test dialog with response preview |
| CLNT-07 | Account settings: profile, DID pool view, security, notification settings | Settings form with shadcn Form components; read-only DID list; password change dialog |
| WAST-04 | Customer can view own wastage rate, detail, B-leg failure distribution | Same as CLNT-05 -- wastage analysis features serve this requirement |
| UI-04 | WebSocket real-time call status push to Admin/Client Dashboard, no polling | gorilla/websocket backend in callback service raw endpoint; frontend useCallStatusWebSocket hook updating TanStack Query cache; Admin WS upgrade |
</phase_requirements>

## Standard Stack

### Core (Frontend -- identical to Phase 4 Admin)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| react | 18.x | UI framework | Locked; matches Admin |
| vite | 6.x | Build tool | Latest stable; matches Admin |
| typescript | 5.x | Type safety | Required for Encore client integration |
| tailwindcss | 4.x | Utility CSS | v4 with `@tailwindcss/vite` plugin |
| shadcn/ui | latest (CLI v4) | Component library | Copy-paste, unified radix-ui package |
| react-router | 7.x | Client routing | Library mode with createBrowserRouter |
| @tanstack/react-query | 5.x | Server state management | Pairs with Encore client + WebSocket invalidation |
| @tanstack/react-table | 8.x | Table/DataTable logic | Required by shadcn DataTable pattern |
| recharts | 3.x | Charts/visualization | Recharts v3, React + D3 based |

### Supporting (Frontend)
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| framer-motion (motion) | 11.x | Animation engine | MagicUI components (NumberTicker, etc.) |
| xlsx (sheetjs) | 0.20.x | Excel export | CDR export to Excel (CLNT-03) |
| date-fns | 3.x | Date formatting | CDR timestamps, transaction dates |
| lucide-react | latest | Icons | shadcn/ui default icon set |
| sonner | latest | Toast notifications | Action feedback toasts |
| zod | latest | Schema validation | Form validation (callback form, settings) |
| react-hook-form | latest | Form management | Callback form, webhook config, account settings |
| @hookform/resolvers | latest | Zod resolver | Connects Zod schemas to react-hook-form |

### Backend Addition (WebSocket)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| gorilla/websocket | 1.5.x | WebSocket server | Battle-tested, most widely used Go WS library; works in Encore raw endpoints |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| gorilla/websocket | coder/websocket (nhooyr) | coder/websocket is more idiomatic Go with context support, but gorilla is more battle-tested and has better docs for the Hub pattern needed here |
| Custom WS hook | react-use-websocket (4.13) | Library adds overhead for a simple use case; a 50-line custom hook with reconnect logic is cleaner and avoids dependency |
| SheetJS for export | CSV string + Blob | SheetJS handles proper Excel formatting (column widths, headers) that raw CSV cannot |

**Installation (Portal):**
```bash
# In /portal directory -- clone admin scaffold then install
npm create vite@latest . -- --template react-ts
npm install react-router @tanstack/react-query @tanstack/react-table recharts
npm install framer-motion date-fns sonner xlsx zod react-hook-form @hookform/resolvers
npm install -D @tailwindcss/vite @types/papaparse

# Initialize shadcn/ui (same config as admin)
npx shadcn@latest init
npx shadcn@latest add button card dialog sheet alert-dialog input label select table badge skeleton sidebar dropdown-menu separator tooltip popover calendar tabs scroll-area avatar switch form toast
```

**Installation (Backend -- WebSocket):**
```bash
go get github.com/gorilla/websocket
```

## Architecture Patterns

### Recommended Project Structure
```
portal/
  src/
    app/                    # App shell, providers, router
      providers.tsx         # QueryClient, ThemeProvider, RouterProvider
      router.tsx            # createBrowserRouter route definitions
      layout.tsx            # Sidebar + TopNav + WS connection banner
    components/
      ui/                   # shadcn/ui components (auto-generated)
      shared/               # Reusable composed components (clone from admin)
        data-table.tsx      # Generic DataTable wrapper
        stat-card.tsx       # Metric card component
        chart-wrapper.tsx   # Recharts theme/config wrapper
        confirm-dialog.tsx  # Danger action confirmation
        skeleton-page.tsx   # Full-page skeleton loader
    features/
      dashboard/            # CLNT-01: Overview
      callback/             # CLNT-02: Callback operations + live calls
      cdr/                  # CLNT-03: CDR query/export/detail
      wastage/              # CLNT-05/WAST-04: Wastage analysis
      finance/              # CLNT-04: Balance, transactions, rates
      api-integration/      # CLNT-06: API Key, Webhook, IP whitelist
      settings/             # CLNT-07: Account settings
      auth/                 # Login page
    hooks/
      use-call-ws.ts        # WebSocket hook for call status updates
    lib/
      api/
        client.ts           # Encore generated client
        hooks.ts            # TanStack Query hooks wrapping client
        error.ts            # Error code to Chinese message mapping
      theme/
        colors.ts           # Vercel-style color palette tokens
        chart-theme.ts      # Recharts shared config
      export.ts             # Excel/CSV export utilities
      utils.ts              # cn() and helpers
    styles/
      globals.css           # Tailwind import + CSS variables
  index.html
  vite.config.ts
  tsconfig.json
  package.json
```

### Backend WebSocket Structure (in callback service)
```
callback/
  ws_hub.go               # Hub: manages connections per user_id
  ws_handler.go           # Encore raw endpoint: upgrade + auth + read/write pump
  ws_message.go           # Message types for call status events
```

### Pattern 1: WebSocket Hub (Backend - Go)
**What:** Central hub managing WebSocket connections by user_id, broadcasting call status events to the correct clients.
**When to use:** The single WebSocket endpoint for both Admin and Client dashboards.
**Example:**
```go
// callback/ws_hub.go
type Hub struct {
    mu          sync.RWMutex
    clients     map[string]map[*Client]bool // user_id -> set of clients
    adminConns  map[*Client]bool            // admin connections (receive all)
    broadcast   chan *CallStatusEvent
    register    chan *Client
    unregister  chan *Client
}

type Client struct {
    hub      *Hub
    conn     *websocket.Conn
    send     chan []byte
    userID   string
    isAdmin  bool
}

type CallStatusEvent struct {
    CallID    string `json:"call_id"`
    UserID    string `json:"user_id"`
    Status    string `json:"status"`
    ALeg      string `json:"a_leg"`
    BLeg      string `json:"b_leg"`
    Duration  int    `json:"duration_sec"`
    Timestamp int64  `json:"timestamp"`
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            if client.isAdmin {
                h.adminConns[client] = true
            } else {
                if h.clients[client.userID] == nil {
                    h.clients[client.userID] = make(map[*Client]bool)
                }
                h.clients[client.userID][client] = true
            }
            h.mu.Unlock()
        case client := <-h.unregister:
            // remove from appropriate map, close send channel
        case event := <-h.broadcast:
            data, _ := json.Marshal(event)
            h.mu.RLock()
            // Send to all admin connections
            for client := range h.adminConns {
                select {
                case client.send <- data:
                default:
                    close(client.send)
                    delete(h.adminConns, client)
                }
            }
            // Send to user's connections
            if clients, ok := h.clients[event.UserID]; ok {
                for client := range clients {
                    select {
                    case client.send <- data:
                    default:
                        close(client.send)
                        delete(clients, client)
                    }
                }
            }
            h.mu.RUnlock()
        }
    }
}
```

### Pattern 2: WebSocket Encore Raw Endpoint (Backend)
**What:** Encore raw endpoint that upgrades HTTP to WebSocket, validates JWT, and registers with Hub.
**When to use:** Single endpoint serving both Admin and Client WebSocket connections.
**Example:**
```go
// callback/ws_handler.go
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true // configure per environment
    },
}

//encore:api public raw path=/ws/calls
func (s *Service) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
    // Extract JWT from query param (?token=xxx) since WebSocket
    // upgrade can't use Authorization header from browser
    token := r.URL.Query().Get("token")
    userID, isAdmin, err := s.validateWSToken(token)
    if err != nil {
        http.Error(w, "unauthorized", http.StatusUnauthorized)
        return
    }

    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }

    client := &Client{
        hub:     s.hub,
        conn:    conn,
        send:    make(chan []byte, 256),
        userID:  userID,
        isAdmin: isAdmin,
    }
    s.hub.register <- client

    go client.writePump()
    go client.readPump() // handles ping/pong and cleanup
}
```

### Pattern 3: Frontend WebSocket Hook + TanStack Query Integration
**What:** Custom React hook that connects to the call status WebSocket, handles reconnection with exponential backoff, and updates TanStack Query cache on incoming events.
**When to use:** Both Portal (callback page, dashboard) and Admin (live call monitoring).
**Example:**
```typescript
// hooks/use-call-ws.ts
import { useEffect, useRef, useCallback, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';

interface CallStatusEvent {
  call_id: string;
  user_id: string;
  status: string;
  a_leg: string;
  b_leg: string;
  duration_sec: number;
  timestamp: number;
}

export function useCallStatusWebSocket() {
  const queryClient = useQueryClient();
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectAttempt = useRef(0);
  const [isConnected, setIsConnected] = useState(false);

  const connect = useCallback(() => {
    const token = localStorage.getItem('token');
    if (!token) return;

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws/calls?token=${token}`);

    ws.onopen = () => {
      setIsConnected(true);
      reconnectAttempt.current = 0;
    };

    ws.onmessage = (event) => {
      const data: CallStatusEvent = JSON.parse(event.data);

      // Update active calls query cache directly
      queryClient.setQueryData<CallStatusEvent[]>(
        ['calls', 'active'],
        (old = []) => {
          if (data.status === 'finished' || data.status === 'failed') {
            return old.filter(c => c.call_id !== data.call_id);
          }
          const idx = old.findIndex(c => c.call_id === data.call_id);
          if (idx >= 0) {
            const updated = [...old];
            updated[idx] = data;
            return updated;
          }
          return [data, ...old];
        }
      );

      // Invalidate dashboard overview for metric updates
      if (data.status === 'finished' || data.status === 'failed') {
        queryClient.invalidateQueries({ queryKey: ['dashboard'] });
      }
    };

    ws.onclose = () => {
      setIsConnected(false);
      // Exponential backoff: 1s, 2s, 4s, 8s, 16s, max 30s
      const delay = Math.min(1000 * Math.pow(2, reconnectAttempt.current), 30000);
      reconnectAttempt.current++;
      setTimeout(connect, delay);
    };

    wsRef.current = ws;
  }, [queryClient]);

  useEffect(() => {
    connect();
    return () => wsRef.current?.close();
  }, [connect]);

  return { isConnected };
}
```

### Pattern 4: CDR Excel/CSV Export
**What:** Client-side export of CDR data to Excel or CSV format.
**When to use:** CLNT-03 CDR export button.
**Example:**
```typescript
// lib/export.ts
import * as XLSX from 'xlsx';

interface CDRRow {
  call_id: string;
  a_number: string;
  b_number: string;
  status: string;
  duration: number;
  cost: number;
  created_at: string;
}

export function exportToExcel(data: CDRRow[], filename: string) {
  const ws = XLSX.utils.json_to_sheet(data);
  // Set column widths
  ws['!cols'] = [
    { wch: 20 }, // call_id
    { wch: 15 }, // a_number
    { wch: 15 }, // b_number
    { wch: 12 }, // status
    { wch: 10 }, // duration
    { wch: 10 }, // cost
    { wch: 20 }, // created_at
  ];
  const wb = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(wb, ws, 'CDR');
  XLSX.writeFile(wb, `${filename}.xlsx`);
}

export function exportToCSV(data: CDRRow[], filename: string) {
  const ws = XLSX.utils.json_to_sheet(data);
  const csv = XLSX.utils.sheet_to_csv(ws);
  const blob = new Blob(['\ufeff' + csv], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `${filename}.csv`;
  a.click();
  URL.revokeObjectURL(url);
}
```

### Pattern 5: Recording Playback (HTML5 Audio)
**What:** Simple audio player in CDR detail side drawer using HTML5 `<audio>` element with presigned S3 URL.
**When to use:** CLNT-03 CDR detail sheet for playing A-leg, B-leg, or merged recordings.
**Example:**
```typescript
// features/cdr/recording-player.tsx
interface RecordingPlayerProps {
  url: string | null;
  label: string;
}

export function RecordingPlayer({ url, label }: RecordingPlayerProps) {
  if (!url) return <p className="text-sm text-muted-foreground">No recording available</p>;

  return (
    <div className="space-y-1">
      <label className="text-sm font-medium">{label}</label>
      <audio controls className="w-full h-8" preload="none">
        <source src={url} type="audio/mpeg" />
        Your browser does not support audio playback.
      </audio>
    </div>
  );
}
// Usage in CDR detail Sheet:
// <RecordingPlayer url={cdr.recording_a_url} label="A 路录音" />
// <RecordingPlayer url={cdr.recording_b_url} label="B 路录音" />
// <RecordingPlayer url={cdr.recording_merged_url} label="合并录音" />
```

### Pattern 6: JWT Token Auth (Portal vs Admin)
**What:** Portal uses Bearer token in localStorage (not HttpOnly cookie like Admin).
**When to use:** All Portal API calls and WebSocket connection.
**Example:**
```typescript
// lib/api/client.ts -- Portal version
// The Encore generated client needs a custom fetch wrapper to attach Bearer token

export function createAuthenticatedClient() {
  const baseUrl = '';  // same-origin via Vite proxy
  const client = new Client(baseUrl);

  // Override the default fetch to include Authorization header
  // This depends on Encore client's generated structure
  // Alternative: use a fetch interceptor
  return client;
}

// For the Encore TS client, set a global fetch wrapper:
const originalFetch = window.fetch;
window.fetch = (input, init) => {
  const token = localStorage.getItem('token');
  if (token) {
    init = init || {};
    init.headers = {
      ...init.headers,
      Authorization: `Bearer ${token}`,
    };
  }
  return originalFetch(input, init);
};
```

### Anti-Patterns to Avoid
- **Polling for call status:** Do NOT use TanStack Query refetchInterval for active call data. WebSocket provides real-time updates. Only use polling for aggregate metrics (overview stats).
- **WebSocket for everything:** Do NOT push overview metrics, CDR lists, or finance data through WebSocket. These are read-heavy, infrequently changing data -- TanStack Query with periodic refetch is correct.
- **Shared code between /admin and /portal via symlinks:** Do NOT symlink shared components. Copy them. The projects will diverge; shared code creates coupling headaches.
- **Storing JWT in cookies for Portal:** The decision is localStorage + Bearer header. Do not mix with Admin's cookie approach.
- **WebSocket reconnect without backoff:** Always use exponential backoff. Reconnecting immediately on close creates thundering herd problems.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| API client | Custom fetch wrapper | `encore gen client --lang=typescript` | Auto-syncs with backend, full type safety |
| Excel export | Manual CSV string building | SheetJS (xlsx) | Handles column widths, Unicode, multiple sheets, BOM for Chinese characters |
| Data tables | Custom table component | shadcn DataTable + TanStack Table | Pagination, sorting, column definitions built-in |
| WebSocket server | net/http raw WS handling | gorilla/websocket | Handles upgrade, ping/pong, close handshake, buffer management |
| Audio player | Custom audio controls | HTML5 `<audio>` element | Native browser controls, no library needed for basic play/pause/seek |
| Form validation | Manual if/else | react-hook-form + Zod | Schema-based, integrates with shadcn Form |
| Toast notifications | Custom system | sonner | Stacking, auto-dismiss, promise-based |
| Theme switching | Custom toggle | shadcn ThemeProvider pattern | System preference, localStorage, no flash |
| Reconnect logic | Complex state machine | Simple exponential backoff in useEffect | 15 lines of code; libraries add unnecessary complexity |

**Key insight:** Portal reuses 80%+ of Admin's patterns. The new work is WebSocket (backend + frontend), CDR export, and recording playback -- all of which have simple, well-established solutions that don't need libraries.

## Common Pitfalls

### Pitfall 1: WebSocket Auth via Query Parameter Exposure
**What goes wrong:** JWT token in WebSocket URL (`?token=xxx`) appears in server access logs and browser history.
**Why it happens:** WebSocket API cannot set custom headers during the upgrade handshake from browsers.
**How to avoid:** Use short-lived tokens specifically for WS connection (not the main auth token), or accept this tradeoff for v1 since it's same-origin. Ensure server logs don't log query parameters. The token in URL is a known pattern used by most WS implementations.
**Warning signs:** Tokens appearing in Nginx/load balancer access logs.

### Pitfall 2: WebSocket Connection Leak on Component Unmount
**What goes wrong:** WebSocket stays open after navigating away from a page, accumulating connections.
**Why it happens:** Forgetting cleanup in useEffect, or reconnect timer firing after unmount.
**How to avoid:** Always return cleanup function from useEffect that calls `ws.close()`. Clear reconnect timeouts on unmount. Keep the WebSocket hook at the app shell level (layout.tsx), not in individual pages.
**Warning signs:** Multiple WebSocket connections visible in browser DevTools Network tab.

### Pitfall 3: TanStack Query Cache Stale After WebSocket Update
**What goes wrong:** WebSocket updates the active calls cache, but other components still show stale data.
**Why it happens:** Using `setQueryData` for the active calls list but forgetting to `invalidateQueries` for related queries (dashboard metrics, CDR lists).
**How to avoid:** On call completion events (finished/failed), invalidate dashboard and CDR query keys. Only use `setQueryData` for the active calls list that needs instant updates.
**Warning signs:** Dashboard counters don't update when calls end, requiring manual page refresh.

### Pitfall 4: Portal JWT vs Admin Cookie Auth Confusion
**What goes wrong:** Portal API calls fail with 401 because the auth handler expects cookies, not Bearer tokens.
**Why it happens:** Phase 1 auth handler supports both modes (cookie for admin, Bearer token for clients), but the Encore-generated client may not include the Authorization header by default.
**How to avoid:** Wrap `window.fetch` with a global interceptor that adds `Authorization: Bearer <token>` header. Ensure the Encore auth handler checks both cookie and Bearer header.
**Warning signs:** Login succeeds but subsequent API calls return 401.

### Pitfall 5: Excel Export Memory for Large CDR Datasets
**What goes wrong:** Exporting 10,000+ CDR records freezes the browser tab.
**Why it happens:** SheetJS processes all data in memory on the main thread.
**How to avoid:** Limit client-side export to the current page/filter results (e.g., max 5000 rows). For larger exports, add a backend endpoint that generates the file server-side and returns a download URL. Show a warning when the dataset exceeds the limit.
**Warning signs:** Browser tab becomes unresponsive during export.

### Pitfall 6: Vite Proxy for Both HTTP API and WebSocket
**What goes wrong:** WebSocket connections fail in development because Vite proxy doesn't handle WS upgrade.
**Why it happens:** Default Vite proxy config doesn't enable WebSocket proxying.
**How to avoid:** Add `ws: true` to the proxy configuration:
```typescript
server: {
  proxy: {
    '/api': { target: 'http://localhost:12345', changeOrigin: true },
    '/ws': { target: 'http://localhost:12345', changeOrigin: true, ws: true },
  }
}
```
**Warning signs:** WebSocket connection fails with 404 or upgrade error in dev mode.

## Code Examples

### WebSocket Connection Banner Component
```typescript
// components/shared/ws-status-banner.tsx
import { AlertTriangle, Wifi } from 'lucide-react';

export function WSStatusBanner({ isConnected }: { isConnected: boolean }) {
  if (isConnected) return null;

  return (
    <div className="bg-destructive/10 border-b border-destructive/20 px-4 py-2 flex items-center gap-2 text-sm text-destructive">
      <AlertTriangle className="h-4 w-4" />
      <span>Real-time connection lost. Reconnecting...</span>
      {/* Chinese: 实时连接已断开，正在重连... */}
    </div>
  );
}
```

### Portal Sidebar Navigation Configuration
```typescript
// app/nav-config.ts
import {
  LayoutDashboard, PhoneCall, FileText,
  TrendingDown, Wallet, Code, Settings
} from 'lucide-react';

export const NAV_ITEMS = [
  {
    group: '概览',
    items: [
      { label: '仪表盘', icon: LayoutDashboard, path: '/' },
    ],
  },
  {
    group: '业务',
    items: [
      { label: '回拨操作', icon: PhoneCall, path: '/callback' },
      { label: '话单查询', icon: FileText, path: '/cdr' },
      { label: '损耗分析', icon: TrendingDown, path: '/wastage' },
    ],
  },
  {
    group: '账务',
    items: [
      { label: '财务中心', icon: Wallet, path: '/finance' },
    ],
  },
  {
    group: '开发',
    items: [
      { label: 'API 集成', icon: Code, path: '/api-integration' },
    ],
  },
  {
    group: '设置',
    items: [
      { label: '账户设置', icon: Settings, path: '/settings' },
    ],
  },
];
```

### Active Call Card Component
```typescript
// features/callback/call-card.tsx
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { PhoneOff } from 'lucide-react';

const STATUS_COLORS: Record<string, string> = {
  initiating: 'bg-gray-500',
  a_dialing: 'bg-yellow-500',
  a_connected: 'bg-blue-500',
  b_dialing: 'bg-cyan-500',
  bridged: 'bg-green-500',
};

const STATUS_LABELS: Record<string, string> = {
  initiating: '发起中',
  a_dialing: 'A路拨号',
  a_connected: 'A路已接',
  b_dialing: 'B路拨号',
  bridged: '通话中',
};

interface CallCardProps {
  callId: string;
  aNumber: string;
  bNumber: string;
  status: string;
  durationSec: number;
  onHangup: (callId: string) => void;
}

export function CallCard({ callId, aNumber, bNumber, status, durationSec, onHangup }: CallCardProps) {
  return (
    <Card>
      <CardContent className="flex items-center justify-between p-4">
        <div className="flex items-center gap-4">
          <Badge className={STATUS_COLORS[status]}>
            {STATUS_LABELS[status] || status}
          </Badge>
          <div>
            <p className="font-mono text-sm">{aNumber} → {bNumber}</p>
            <p className="text-xs text-muted-foreground">
              {Math.floor(durationSec / 60)}:{String(durationSec % 60).padStart(2, '0')}
            </p>
          </div>
        </div>
        <Button variant="destructive" size="sm" onClick={() => onHangup(callId)}>
          <PhoneOff className="h-4 w-4 mr-1" />
          挂断
        </Button>
      </CardContent>
    </Card>
  );
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| gorilla/websocket only | gorilla/websocket or coder/websocket | 2024 | coder/websocket is the maintained fork of nhooyr; gorilla remains stable |
| react-router-dom v6 | react-router v7 (unified) | 2024 Q4 | Single import package |
| tailwind.config.js | CSS-first @theme directive | Tailwind v4, 2025 Q1 | No JS config file |
| Individual @radix-ui packages | Unified radix-ui | shadcn 2026-02 | Single dependency |
| WebSocket + Redux | WebSocket + TanStack Query cache invalidation | 2024-2025 | Simpler state, no Redux needed |

**Deprecated/outdated:**
- `react-router-dom`: Merged into `react-router` v7
- `tailwind.config.js`: Replaced by CSS `@theme` directive in v4
- Socket.IO for simple WS: Overkill for this use case; native WebSocket + gorilla is sufficient

## Open Questions

1. **Encore Generated Client + Bearer Token**
   - What we know: Encore generates a TypeScript client. Admin uses cookies. Portal needs Bearer tokens.
   - What's unclear: Whether the generated client supports custom headers out of the box or needs a fetch wrapper.
   - Recommendation: Wrap `window.fetch` globally before creating the client instance. This is the simplest approach and doesn't require modifying generated code.

2. **WebSocket Token Validation Scope**
   - What we know: JWT token is passed as query param on WS connect. The auth handler validates JWT.
   - What's unclear: Whether to reuse the same JWT or issue a shorter-lived WS-specific token.
   - Recommendation: Reuse the main JWT for v1 simplicity. The token is already same-origin. Add WS-specific tokens in v2 if security audit requires it.

3. **Admin WebSocket Upgrade Scope**
   - What we know: Admin Dashboard currently uses 30s polling (Phase 4). Phase 5 upgrades it to WebSocket.
   - What's unclear: How much of the Admin code needs changing -- just the live call monitoring page, or the overview dashboard too?
   - Recommendation: Add WebSocket to Admin's live call monitoring only (replacing polling for active calls). Overview metrics can remain on TanStack Query refetchInterval since they're aggregate data.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework (Frontend) | Vitest 3.x (Vite-native test runner) |
| Framework (Backend) | encore test (Go test with Encore extensions) |
| Config file (Frontend) | portal/vitest.config.ts (Wave 0) |
| Quick run command (Frontend) | `cd portal && npx vitest run --reporter=verbose` |
| Full suite command (Frontend) | `cd portal && npx vitest run --coverage` |
| Quick run command (Backend) | `encore test ./callback/... -run TestWebSocket -count=1` |

### Phase Requirements -> Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| CLNT-01 | Dashboard renders metric cards and trend charts | unit | `cd portal && npx vitest run src/features/dashboard/ -x` | Wave 0 |
| CLNT-02 | Callback form validates and submits; call cards render with status | unit | `cd portal && npx vitest run src/features/callback/ -x` | Wave 0 |
| CLNT-03 | CDR table renders, export produces file, detail sheet shows recording player | unit | `cd portal && npx vitest run src/features/cdr/ -x` | Wave 0 |
| CLNT-04 | Finance page shows balance, transactions table, rate query | unit | `cd portal && npx vitest run src/features/finance/ -x` | Wave 0 |
| CLNT-05 | Wastage charts render with mock data | unit | `cd portal && npx vitest run src/features/wastage/ -x` | Wave 0 |
| CLNT-06 | API key display, webhook config form, IP whitelist CRUD | unit | `cd portal && npx vitest run src/features/api-integration/ -x` | Wave 0 |
| CLNT-07 | Account settings form renders and validates | unit | `cd portal && npx vitest run src/features/settings/ -x` | Wave 0 |
| WAST-04 | Same as CLNT-05 | unit | Same as CLNT-05 | Wave 0 |
| UI-04 | WebSocket hook connects, handles events, updates query cache | unit | `cd portal && npx vitest run src/hooks/ -x` | Wave 0 |
| UI-04 (backend) | Hub registers/unregisters clients, broadcasts to correct users | unit | `encore test ./callback/... -run TestHub -count=1` | Wave 0 |

### Sampling Rate
- **Per task commit:** `cd portal && npx vitest run --reporter=verbose`
- **Per wave merge:** `cd portal && npx vitest run --coverage`
- **Phase gate:** Full suite green + backend WS tests pass before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `portal/vitest.config.ts` -- Vitest configuration with jsdom environment
- [ ] `portal/src/test/setup.ts` -- Test setup (jsdom, mock matchMedia, mock ResizeObserver)
- [ ] `portal/src/test/mocks/api.ts` -- Mock Encore client for unit tests
- [ ] `portal/src/test/mocks/websocket.ts` -- Mock WebSocket for WS hook tests
- [ ] Install: `npm install -D vitest @testing-library/react @testing-library/jest-dom jsdom @testing-library/user-event`
- [ ] `callback/ws_hub_test.go` -- Hub unit tests (register, unregister, broadcast routing)

## Sources

### Primary (HIGH confidence)
- [Encore raw endpoints docs](https://encore.dev/docs/go/primitives/raw-endpoints) - WebSocket support via raw endpoints
- [Encore frontend integration docs](https://encore.dev/docs/go/how-to/integrate-frontend) - Client generation, CORS
- [gorilla/websocket GitHub](https://github.com/gorilla/websocket) - Hub pattern in chat example, v1.5.x API
- [shadcn/ui docs](https://ui.shadcn.com/) - Component library, DataTable, Sidebar, Form patterns
- [TanStack Query + WebSockets by TkDodo](https://tkdodo.eu/blog/using-web-sockets-with-react-query) - Cache invalidation strategy for WS events
- [SheetJS docs](https://docs.sheetjs.com/docs/solutions/output/) - Excel export API
- Phase 4 Research (04-RESEARCH.md) - Verified stack versions, architecture patterns, all reusable for Phase 5

### Secondary (MEDIUM confidence)
- [react-use-websocket npm](https://www.npmjs.com/package/react-use-websocket) - v4.13.0, evaluated but custom hook preferred
- [coder/websocket GitHub](https://github.com/coder/websocket) - Evaluated as alternative to gorilla, not selected

### Tertiary (LOW confidence)
- None -- all findings verified with primary sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Identical to Phase 4 Admin (already verified), only additions are gorilla/websocket (well-established) and SheetJS (stable)
- Architecture: HIGH - Hub pattern is the canonical gorilla/websocket example; TanStack Query + WS invalidation is the TkDodo-recommended approach
- Pitfalls: HIGH - WS auth via query param, Vite WS proxy, connection leaks are all well-documented common issues
- WebSocket integration: HIGH - Pattern verified via TkDodo blog (TanStack Query maintainer's official recommendation)

**Research date:** 2026-03-10
**Valid until:** 2026-04-10 (stable ecosystem, 30 days)
