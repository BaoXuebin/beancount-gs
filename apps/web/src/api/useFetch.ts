import { useCallback, useEffect, useState } from 'react'
import { request, type RequestOptions } from './client'

export interface FetchState<T> {
  data: T | null
  error: string | null
  loading: boolean
  refetch: () => void
}

export function useFetch<T>(path: string, opts: RequestOptions = {}): FetchState<T> {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const refetch = useCallback(() => {
    setLoading(true)
    setError(null)
    request<T>(path, opts)
      .then(setData)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : String(err)))
      .finally(() => setLoading(false))
  }, [path])

  useEffect(() => {
    refetch()
  }, [refetch])

  return { data, error, loading, refetch }
}
