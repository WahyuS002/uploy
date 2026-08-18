import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { createApiClient } from '$lib/api/client';
import type { components } from '$lib/api/v1';
import { env } from '$env/dynamic/private';

export const load: PageServerLoad = async ({ fetch }) => {
	const api = createApiClient(fetch, env.API_BASE_URL);
	const [serversRes, keysRes] = await Promise.all([
		api.GET('/api/servers'),
		api.GET('/api/ssh-keys')
	]);
	if (serversRes.error) {
		throw error(500, 'Failed to load servers');
	}
	if (keysRes.error) {
		throw error(500, 'Failed to load SSH keys');
	}

	type ServerHealth = {
		latest: components['schemas']['ServerObservabilityResponse'] | null;
		history: components['schemas']['ServerObservabilityResponse'][];
	};
	const healthEntries = await Promise.all(
		serversRes.data.map(async (server) => {
			if (!server.monitoring.enabled) {
				return [server.id, null] as const;
			}
			const [latestRes, historyRes] = await Promise.all([
				api.GET('/api/servers/{id}/observability', { params: { path: { id: server.id } } }),
				api.GET('/api/servers/{id}/observability/history', {
					params: { path: { id: server.id }, query: { since: '24h', max_points: 48 } }
				})
			]);
			return [
				server.id,
				{
					latest: latestRes.error ? null : latestRes.data,
					history: historyRes.error ? [] : historyRes.data.points
				}
			] as const;
		})
	);

	return {
		servers: serversRes.data,
		keys: keysRes.data,
		serverHealth: Object.fromEntries(healthEntries) as Record<string, ServerHealth | null>
	};
};
