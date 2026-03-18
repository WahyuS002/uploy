<script lang="ts">
	let email = $state('');
	let password = $state('');
	let error = $state('');
	let loading = $state(false);

	async function handleLogin() {
		error = '';
		loading = true;

		try {
			const res = await fetch('/api/auth/login', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ email, password })
			});

			if (!res.ok) {
				const data = await res.json();
				error = data.error || 'Login failed';
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
		<h1 class="mb-6 text-2xl font-bold">Sign in to Uploy</h1>

		{#if error}
			<div class="mb-4 rounded bg-red-50 p-3 text-sm text-red-600">{error}</div>
		{/if}

		<form
			onsubmit={(e) => {
				e.preventDefault();
				handleLogin();
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
				/>
			</label>

			<button
				type="submit"
				disabled={loading}
				class="cursor-pointer rounded bg-black p-2 text-white disabled:opacity-50"
			>
				{loading ? 'Signing in...' : 'Sign in'}
			</button>
		</form>

		<p class="mt-4 text-center text-sm text-gray-600">
			Don't have an account? <a href="/register" class="text-black underline">Register</a>
		</p>
	</div>
</div>
