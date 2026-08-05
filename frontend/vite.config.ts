import tailwindcss from '@tailwindcss/vite';
import { sveltekit } from '@sveltejs/kit/vite';
import Icons from 'unplugin-icons/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	// Brand logos are inlined at build time from @iconify-json/logos (CC0). No CDN
	// call at runtime: this dashboard has to work on a box with no egress.
	plugins: [tailwindcss(), Icons({ compiler: 'svelte' }), sveltekit()],
	server: {
		proxy: {
			'/api': 'http://localhost:8080'
		}
	}
});
