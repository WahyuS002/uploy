import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { createApiClient } from '$lib/api/client';
import { env } from '$env/dynamic/private';

export const load: PageServerLoad = async ({ fetch }) => {
	const api = createApiClient(fetch, env.API_BASE_URL);
	const { data, error: err } = await api.GET('/api/ssh-keys');
	if (err) {
		throw error(500, 'Failed to load SSH keys');
	}
	return { keys: data };
};
