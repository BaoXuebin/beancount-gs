import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError, request, type RequestOptions } from './client'

export interface FetchState<T> {
  data: T | null
  error: string | null
  errorStatus?: number | null
  errorCode?: string | null
  loading: boolean
  refetch: () => void
}

export function useFetch<T>(path: string, opts: RequestOptions = {}, reloadKey?: unknown): FetchState<T> {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [errorStatus, setErrorStatus] = useState<number | null>(null)
  const [errorCode, setErrorCode] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const optsRef = useRef(opts)
  optsRef.current = opts

  const refetch = useCallback(() => {
    setLoading(true)
    setError(null)
    setErrorStatus(null)
    setErrorCode(null)
    request<T>(path, optsRef.current)
      .then(setData)
      .catch((err: unknown) => {
        if (err instanceof ApiError) {
          setError(err.message)
          setErrorStatus(err.status)
          setErrorCode(err.code)
        } else {
          setError(err instanceof Error ? err.message : String(err))
        }
      })
      .finally(() => setLoading(false))
  }, [path, reloadKey])

  useEffect(() => {
    refetch()
  }, [refetch])

  return { data, error, errorStatus, errorCode, loading, refetch }
}
