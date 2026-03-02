<script lang="ts">
	let output = 'Klik button diatas untuk pull nginx:latest';

	async function fetchDockerNginx() {
		const res = await fetch('/api/docker/nginx');
		output = await res.text();
	}
</script>

<h1>Hello World (uploy)</h1>

<button class="cursor-pointer rounded-md bg-black px-2 py-2 text-white" onclick={fetchDockerNginx}>Deploy Nginx</button>
<p>{output}</p>
