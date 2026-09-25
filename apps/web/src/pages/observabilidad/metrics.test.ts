import { describe, expect, it } from 'vitest'
import { readMetricValue } from './metrics'

describe('readMetricValue', () => {
	it('reads a counter from Prometheus text', () => {
		const text = `# TYPE http_requests_total counter
http_requests_total{method="GET",route="/portal",status="200"} 7
http_requests_total{method="POST",route="/login",status="401"} 2
`
		expect(readMetricValue(text, 'http_requests_total')).toBe(7)
	})

	it('returns null when the metric is missing', () => {
		expect(readMetricValue('# TYPE up gauge\nup 1\n', 'outbox_failed_total')).toBeNull()
	})
})
