import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { createApiClient } from '$lib/api/client';
import { env } from '$env/dynamic/private';

export const load: PageServerLoad = async ({ fetch }) => {
	const api = createApiClient(fetch, env.API_BASE_URL);
	const [channelsRes, rulesRes, historyRes] = await Promise.all([
		api.GET('/api/notification-channels'),
		api.GET('/api/alert-rules'),
		api.GET('/api/alerts/history', { params: { query: { limit: 50, offset: 0 } } })
	]);
	if (channelsRes.error || rulesRes.error || historyRes.error) {
		throw error(500, 'Failed to load alerts');
	}
	return {
		channels: channelsRes.data ?? [],
		rules: rulesRes.data ?? [],
		history: historyRes.data ?? []
	};
};
