import { api } from '$lib/api/client';

// logout sends POST /api/auth/logout and navigates to /login on success.
// A 401 response means the session is already gone — treated as success so a
// stale-session click still clears the client and redirects. Any other error
// (network failure, 5xx) is surfaced to the caller so the UI can re-enable
// the trigger and show feedback.
export async function logout(): Promise<{ ok: true } | { ok: false; message: string }> {
	try {
		const { response, error: err } = await api.POST('/api/auth/logout');
		if (err && response.status !== 401) {
			const message =
				(err as { error?: string } | null)?.error?.trim() || 'Failed to log out. Please try again.';
			return { ok: false, message };
		}
	} catch {
		return { ok: false, message: 'Network error. Please check your connection and try again.' };
	}
	window.location.href = '/login';
	return { ok: true };
}
