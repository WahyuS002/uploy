import type { PageServerLoad } from './$types';
import { createApiClient } from '$lib/api/client';
import { env } from '$env/dynamic/private';

export const load: PageServerLoad = async ({ fetch }) => {
	const api = createApiClient(fetch, env.API_BASE_URL);
	const { data, error } = await api.GET('/api/servers');
	// Same contract as ../+page.server.ts: a failed list is reported, not thrown,
	// so the page can render its own message with a retry.
	if (error) {
		const message = (error as { error?: string }).error ?? 'Failed to load servers';
		return { servers: [], serversError: message };
	}
	return { servers: data ?? [], serversError: '' };
};
