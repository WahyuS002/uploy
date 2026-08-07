import { Select } from 'bits-ui';
import Trigger from './select-action/Trigger.svelte';
import Content from './select-action/Content.svelte';
import Item from './select-action/Item.svelte';

export const SelectAction = {
	Root: Select.Root,
	Portal: Select.Portal,
	Viewport: Select.Viewport,
	Trigger,
	Content,
	Item
};
