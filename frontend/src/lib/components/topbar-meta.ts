import type { IconSource } from '@steeze-ui/svelte-icon';
import { Key, Server, Squares2x2 } from '@steeze-ui/heroicons';
import type { components } from '$lib/api/v1';

type ServiceResponse = components['schemas']['ServiceResponse'];

export type TopbarMeta = {
	title: string;
	icon?: IconSource;
};

type RouteData =
	| {
			service?: ServiceResponse;
	  }
	| null
	| undefined;

const STATIC_META: Record<string, TopbarMeta> = {
	'/(app)/(dashboard)/projects': { title: 'Projects', icon: Squares2x2 },
	'/(app)/(dashboard)/servers': { title: 'Servers', icon: Server },
	'/(app)/(dashboard)/ssh-keys': { title: 'SSH Keys', icon: Key }
};

export function getDashboardTopbarMeta(routeId: string | null, data: RouteData): TopbarMeta {
	if (!routeId) return { title: '' };

	const staticMeta = STATIC_META[routeId];
	if (staticMeta) return staticMeta;

	if (routeId === '/(app)/(dashboard)/services/[id]') {
		return { title: data?.service?.name ?? '' };
	}

	return { title: '' };
}
