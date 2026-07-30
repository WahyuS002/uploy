import type { IconSource } from '@steeze-ui/svelte-icon';
import { Squares2x2, Server, FolderPlus, ServerStack } from '@steeze-ui/heroicons';

export type QuickActionGroup = 'Navigate' | 'Create';

export type QuickAction = {
	id: string;
	label: string;
	keywords: string[];
	group: QuickActionGroup;
	href: string;
	icon: IconSource;
	visibleForRole?: ReadonlyArray<string>;
};

export const QUICK_ACTIONS: ReadonlyArray<QuickAction> = [
	{
		id: 'nav-projects',
		label: 'Projects',
		keywords: ['projects', 'workspace'],
		group: 'Navigate',
		href: '/projects',
		icon: Squares2x2
	},
	{
		id: 'nav-servers',
		label: 'Servers',
		keywords: ['servers', 'machines', 'hosts'],
		group: 'Navigate',
		href: '/servers',
		icon: Server
	},
	{
		id: 'create-project',
		label: 'Create project',
		keywords: ['new', 'project', 'add'],
		group: 'Create',
		href: '/projects/new',
		icon: FolderPlus,
		visibleForRole: ['owner', 'developer']
	},
	{
		id: 'create-server',
		label: 'Add server',
		keywords: ['new', 'server', 'add', 'host', 'ssh', 'key', 'credentials'],
		group: 'Create',
		href: '/servers',
		icon: ServerStack,
		visibleForRole: ['owner']
	}
];

export function filterByRole(
	actions: ReadonlyArray<QuickAction>,
	role: string | undefined
): QuickAction[] {
	return actions.filter(
		(a) => !a.visibleForRole || (role ? a.visibleForRole.includes(role) : false)
	);
}

export function groupActions(
	actions: ReadonlyArray<QuickAction>
): Record<QuickActionGroup, QuickAction[]> {
	const groups: Record<QuickActionGroup, QuickAction[]> = { Navigate: [], Create: [] };
	for (const a of actions) groups[a.group].push(a);
	return groups;
}
