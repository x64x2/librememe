<script lang="ts">
	import { DatePicker } from 'date-picker-svelte'
	import { format as dateFnsFormat } from 'date-fns'
	import { createEventDispatcher } from 'svelte'

	export let onlyVisible: boolean
	export let date: Date | null = null

	const dispatch = createEventDispatcher()

	let showDatepicker = false

	function pickedDate(event: CustomEvent) {
		showDatepicker = false
		dispatch('pickedDate', event.detail)
	}

	function toggleDatepicker() {
		showDatepicker = !showDatepicker
	}
</script>

<div class="flex flex-row space-x-5">
	<label><input type="checkbox" bind:checked={onlyVisible} /> Hide not visible</label>
	<button on:click={toggleDatepicker}>
		{#if date}
			{dateFnsFormat(date, 'yyyy-MM-dd')}
		{:else}
			Jump to day
		{/if}
	</button>
</div>

{#if showDatepicker}
	<DatePicker bind:value={date} browseWithoutSelecting={true} on:select={pickedDate} />
{/if}
