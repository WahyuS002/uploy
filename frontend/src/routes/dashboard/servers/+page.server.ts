import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { createApiClient } from '$lib/api/client';
import { env } from '$env/dynamic/private';

export const load: PageServerLoad = async ({ fetch }) => {
	const api = createApiClient(fetch, env.API_BASE_URL);
	const serversRes = await api.GET('/api/servers');
	if (serversRes.error) {
		throw error(500, 'Failed to load servers');
	}
	return { servers: serversRes.data };
};
