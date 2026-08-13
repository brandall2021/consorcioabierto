import createClient from 'openapi-fetch'
import type { paths } from './generated.d'

export const client = createClient<paths>({
  baseUrl: '/api/v1',
  credentials: 'include',
})
