import type { LayoutServerLoad } from './$types';
import { createApiClient } from '$lib/api/client';
import { env } from '$env/dynamic/private';

// Loaded here, not per page, because /projects/new and /projects/new/image both
// need the same list. A layout load is not re-run when navigating between its
// children, so stepping into the image form costs no round-trip — the step is
// instant instead of stalling on the network before the enter animation starts.
//
// Deliberately not thrown: a server list that fails to load must not take the
// page down. Empty Project needs no server, and each page renders the message
// with a retry of its own.
export const load: LayoutServerLoad = async ({ fetch }) => {
	const api = createApiClient(fetch, env.API_BASE_URL);
	const { data, error } = await api.GET('/api/servers');
	if (error) {
		const message = (error as { error?: string }).error ?? 'Failed to load servers';
		return { servers: [], serversError: message };
	}
	return { servers: data ?? [], serversError: '' };
};
