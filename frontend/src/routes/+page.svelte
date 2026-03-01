<script lang="ts">
	let output = 'Klik button diatas untuk fetch docker ps';

	async function fetchDockerPs() {
		const res = await fetch('/api/docker/ps');
		output = await res.text();
	}
</script>

<h1>Hello World (uploy)</h1>

<button class="cursor-pointer rounded-md bg-black px-2 py-2 text-white" onclick={fetchDockerPs}>Fetch Docker Ps</button>
<p>{output}</p>
