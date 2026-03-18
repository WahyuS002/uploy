import { redirect } from '@sveltejs/kit';
import type { LayoutServerLoad } from './$types';

export const load: LayoutServerLoad = async ({ fetch }) => {
	const res = await fetch('/api/auth/me');
	if (!res.ok) {
		throw redirect(302, '/login');
	}
	return await res.json();
};
