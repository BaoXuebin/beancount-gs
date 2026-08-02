export const API_BASE: string = import.meta.env.VITE_API_BASE ?? '/api/v2'

export interface RequestOptions {
  ledgerId?: string
  revision?: number
  token?: string
}

export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly details?: Record<string, unknown>

  constructor(status: number, code: string, message: string, details?: Record<string, unknown>) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.details = details
  }
}

export interface ApiErrorBody {
  code?: string
  message?: string
  details?: Record<string, unknown>
}

export async function request<T>(
  path: string,
  init: RequestInit & RequestOptions = {},
): Promise<T> {
  const { ledgerId, revision, token, ...rest } = init
  const headers = new Headers(rest.headers)
  if (ledgerId) headers.set('ledger-id', ledgerId)
  if (revision !== undefined) headers.set('If-Revision-Match', String(revision))
  if (token) headers.set('Authorization', `Bearer ${token}`)
  if (rest.body != null && !(rest.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  const res = await fetch(`${API_BASE}${path}`, { ...rest, headers })
  if (!res.ok) {
    let body: ApiErrorBody | undefined
    try {
      body = (await res.json()) as ApiErrorBody
    } catch {
      // 非 JSON 错误响应
    }
    throw new ApiError(
      res.status,
      body?.code ?? 'UNKNOWN',
      body?.message ?? res.statusText,
      body?.details,
    )
  }
  if (res.status === 204) {
    return undefined as T
  }
  return (await res.json()) as T
}

export const api = {
  get<T>(path: string, opts: RequestOptions = {}): Promise<T> {
    return request<T>(path, { method: 'GET', ...opts })
  },
  post<T>(path: string, body?: unknown, opts: RequestOptions = {}): Promise<T> {
    return request<T>(path, {
      method: 'POST',
      body: body === undefined ? undefined : JSON.stringify(body),
      ...opts,
    })
  },
  put<T>(path: string, body?: unknown, opts: RequestOptions = {}): Promise<T> {
    return request<T>(path, {
      method: 'PUT',
      body: body === undefined ? undefined : JSON.stringify(body),
      ...opts,
    })
  },
  delete<T>(path: string, opts: RequestOptions = {}): Promise<T> {
    return request<T>(path, { method: 'DELETE', ...opts })
  },
}
