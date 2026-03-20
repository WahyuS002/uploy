import { redirect } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';
import { createApiClient } from '$lib/api/client';

export const load: LayoutServerLoad = async ({ fetch }) => {
	const api = createApiClient(fetch);
	const { data, error } = await api.GET('/api/auth/me');
	if (error) {
		throw redirect(302, '/login');
	}
	return data;
};
