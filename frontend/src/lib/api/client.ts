import createClient from 'openapi-fetch';
import type { paths } from './v1';

export function createApiClient(customFetch?: typeof fetch, baseUrl = '') {
	return createClient<paths>({
		baseUrl,
		credentials: 'include',
		...(customFetch ? { fetch: customFetch } : {})
	});
}

export const api = createApiClient();
