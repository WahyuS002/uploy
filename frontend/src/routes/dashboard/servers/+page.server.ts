import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { createApiClient } from '$lib/api/client';
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
	return { servers: serversRes.data, keys: keysRes.data };
};
