<script lang="ts">
	import { api } from '$lib/api/client';

	let email = $state('');
	let password = $state('');
	let confirmPassword = $state('');
	let error = $state('');
	let loading = $state(false);

	async function handleRegister() {
		error = '';

		if (password !== confirmPassword) {
			error = 'Passwords do not match';
			return;
		}

		loading = true;

		try {
			const { data, error: err } = await api.POST('/api/auth/register', {
				body: { email, password }
			});

			if (err) {
				error = err.error || 'Registration failed';
				return;
			}

			window.location.href = '/dashboard';
		} catch {
			error = 'Network error';
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex min-h-screen items-center justify-center">
	<div class="w-full max-w-sm">
		<h1 class="mb-6 text-2xl font-bold">Create your account</h1>

		{#if error}
			<div class="mb-4 rounded bg-red-50 p-3 text-sm text-red-600">{error}</div>
		{/if}

		<div class="flex flex-col gap-3">
			<a
				href="/api/auth/github"
				class="flex items-center justify-center gap-2 rounded border p-2 text-sm font-medium hover:bg-gray-50"
			>
				<svg class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor">
					<path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
				</svg>
				Continue with GitHub
			</a>

			<a
				href="/api/auth/google"
				class="flex items-center justify-center gap-2 rounded border p-2 text-sm font-medium hover:bg-gray-50"
			>
				<svg class="h-5 w-5" viewBox="0 0 24 24">
					<path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" fill="#4285F4"/>
					<path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
					<path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
					<path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
				</svg>
				Continue with Google
			</a>
		</div>

		<div class="my-5 flex items-center gap-3">
			<div class="h-px flex-1 bg-gray-200"></div>
			<span class="text-sm text-gray-400">or</span>
			<div class="h-px flex-1 bg-gray-200"></div>
		</div>

		<form
			onsubmit={(e) => {
				e.preventDefault();
				handleRegister();
			}}
			class="flex flex-col gap-4"
		>
			<label class="flex flex-col gap-1 text-sm">
				Email
				<input
					type="email"
					bind:value={email}
					required
					class="rounded border p-2"
					placeholder="you@example.com"
				/>
			</label>

			<label class="flex flex-col gap-1 text-sm">
				Password
				<input
					type="password"
					bind:value={password}
					required
					minlength="8"
					class="rounded border p-2"
					placeholder="Min 8 characters"
				/>
			</label>

			<label class="flex flex-col gap-1 text-sm">
				Confirm Password
				<input
					type="password"
					bind:value={confirmPassword}
					required
					minlength="8"
					class="rounded border p-2"
				/>
			</label>

			<button
				type="submit"
				disabled={loading}
				class="cursor-pointer rounded bg-black p-2 text-white disabled:opacity-50"
			>
				{loading ? 'Creating account...' : 'Create account'}
			</button>
		</form>

		<p class="mt-4 text-center text-sm text-gray-600">
			Already have an account? <a href="/login" class="text-black underline">Sign in</a>
		</p>
	</div>
</div>
