import { toast } from 'sonner'

const ERROR_MESSAGES: Record<string, string> = {
  INSUFFICIENT_BALANCE: '余额不足，请先充值',
  BLACKLISTED_NUMBER: '该号码已被加入黑名单',
  RATE_LIMIT_EXCEEDED: '已达到今日呼叫限额',
  CONCURRENCY_LIMIT_EXCEEDED: '并发数已达上限',
  INVALID_CREDENTIALS: '用户名或密码错误',
  USER_FROZEN: '该账户已被冻结',
  USER_NOT_FOUND: '用户不存在',
  GATEWAY_NOT_FOUND: '网关不存在',
  GATEWAY_DOWN: '网关离线，无法操作',
  DID_NOT_AVAILABLE: '该 DID 号码不可用',
  DID_ALREADY_ASSIGNED: '该 DID 号码已分配',
  DUPLICATE_ENTRY: '记录已存在',
  PERMISSION_DENIED: '权限不足',
  INVALID_ARGUMENT: '参数无效',
  NOT_FOUND: '记录不存在',
  INTERNAL: '系统内部错误，请稍后重试',
  UNAUTHENTICATED: '登录已过期，请重新登录',
  UNAVAILABLE: '服务暂不可用，请稍后重试',
}

export interface ApiError {
  code: string
  message: string
  details?: Record<string, unknown>
}

export function getErrorMessage(err: unknown): string {
  if (err && typeof err === 'object' && 'code' in err) {
    const apiErr = err as ApiError
    return ERROR_MESSAGES[apiErr.code] ?? ERROR_MESSAGES[apiErr.message] ?? apiErr.message ?? '操作失败，请稍后重试'
  }
  if (err instanceof Error) {
    return err.message
  }
  return '操作失败，请稍后重试'
}

export function toastError(err: unknown) {
  toast.error(getErrorMessage(err))
}
