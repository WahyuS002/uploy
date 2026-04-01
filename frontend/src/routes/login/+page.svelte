<script lang="ts">
	import { page } from '$app/state';
	import { api } from '$lib/api/client';
	import AuthCard from '$lib/components/app/AuthCard.svelte';
	import FormField from '$lib/components/app/FormField.svelte';
	import Button from '$lib/components/ui/Button.svelte';
	import Input from '$lib/components/ui/Input.svelte';
	import Alert from '$lib/components/ui/Alert.svelte';

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);

	const oauthErrors: Record<string, string> = {
		oauth_denied: 'OAuth login was denied',
		oauth_failed: 'OAuth login failed, please try again',
		invalid_state: 'Invalid OAuth state, please try again',
		account_inactive: 'Account is not active',
		internal: 'Something went wrong, please try again'
	};

	$effect(() => {
		const errParam = page.url.searchParams.get('error');
		if (errParam) {
			error = oauthErrors[errParam] || errParam;
		}
	});

	async function handleLogin() {
		error = '';
		loading = true;

		try {
			const { error: err } = await api.POST('/api/auth/login', {
				body: { email, password }
			});

			if (err) {
				error = err.error || 'Login failed';
				return;
			}

			window.location.href = '/dashboard/projects';
		} catch {
			error = 'Network error';
		} finally {
			loading = false;
		}
	}
</script>

<AuthCard title="Sign in to Uploy">
	{#if error}
		<Alert tone="danger" class="mb-4">{error}</Alert>
	{/if}

	<div class="flex flex-col gap-3">
		<Button href="/api/auth/github" variant="secondary" class="w-full">
			<svg class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor">
				<path
					d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"
				/>
			</svg>
			Continue with GitHub
		</Button>

		<Button href="/api/auth/google" variant="secondary" class="w-full">
			<svg class="h-5 w-5" viewBox="0 0 24 24">
				<path
					d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z"
					fill="#4285F4"
				/>
				<path
					d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
					fill="#34A853"
				/>
				<path
					d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
					fill="#FBBC05"
				/>
				<path
					d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
					fill="#EA4335"
				/>
			</svg>
			Continue with Google
		</Button>
	</div>

	<div class="my-5 flex items-center gap-3">
		<div class="h-px flex-1 bg-border"></div>
		<span class="text-sm text-muted-foreground">or</span>
		<div class="h-px flex-1 bg-border"></div>
	</div>

	<form
		onsubmit={(e) => {
			e.preventDefault();
			handleLogin();
		}}
		class="flex flex-col gap-4"
	>
		<FormField label="Email">
			<Input type="email" bind:value={email} required placeholder="you@example.com" />
		</FormField>

		<FormField label="Password">
			<Input type="password" bind:value={password} required minlength={8} />
		</FormField>

		<Button type="submit" {loading} class="w-full">
			{loading ? 'Signing in...' : 'Sign in'}
		</Button>
	</form>

	{#snippet footer()}
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
		Don't have an account? <a href="/register" class="text-foreground underline">Register</a>
	{/snippet}
</AuthCard>
