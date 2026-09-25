const NUMBER_RE = '([0-9]+(?:\\.[0-9]+)?)'

function escapeRegex(input: string): string {
	return input.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

export function readMetricValue(text: string, metricName: string): number | null {
	const pattern = new RegExp(`^${escapeRegex(metricName)}(?:\\{[^\\n]*\\})?\\s+${NUMBER_RE}$`, 'm')
	const match = text.match(pattern)
	if (!match) return null
	return Number(match[1])
}
