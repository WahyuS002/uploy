import { error } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';
import { createApiClient } from '$lib/api/client';
import { env } from '$env/dynamic/private';

export const load: PageServerLoad = async ({ fetch, params }) => {
	const api = createApiClient(fetch, env.API_BASE_URL);
	const { data, error: err } = await api.GET('/api/services/{id}', {
		params: { path: { id: params.id } }
	});
	if (err || !data) {
		throw error(404, 'Service not found');
	}
	return { service: data };
};
