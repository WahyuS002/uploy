// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
declare global {
	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		interface PageState {
			toastFlash?: {
				tone?: 'neutral' | 'success' | 'error';
				title: string;
				description?: string;
			};
		}
		// interface Platform {}
	}
}

export {};
