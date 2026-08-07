import { redirect } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';
import { createApiClient } from '$lib/api/client';
import { env } from '$env/dynamic/private';

export const load: LayoutServerLoad = async ({ fetch }) => {
	const api = createApiClient(fetch, env.API_BASE_URL);
	const { data, error } = await api.GET('/api/auth/me');
	if (error) {
		throw redirect(302, '/login');
	}
	return data;
};
