import type { PageServerLoad } from './$types';
import { createApiClient } from '$lib/api/client';
import { env } from '$env/dynamic/private';

export const load: PageServerLoad = async ({ fetch }) => {
	const api = createApiClient(fetch, env.API_BASE_URL);
	const { data, error } = await api.GET('/api/servers');
	// Deliberately not thrown: a server list that fails to load must not take the
	// page down. Empty Project needs no server, and the node renders the message
	// with a retry of its own.
	if (error) {
		const message = (error as { error?: string }).error ?? 'Failed to load servers';
		return { servers: [], serversError: message };
	}
	return { servers: data ?? [], serversError: '' };
};
