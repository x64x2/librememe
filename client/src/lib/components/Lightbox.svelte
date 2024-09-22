<script lang="ts">
	import { createEventDispatcher } from 'svelte'

	export let url: string
	export let type: string

	const dispatcher = createEventDispatcher()

	function close(this: Element, event: Event) {
		if (this != event.target) {
			return
		}
		dispatcher('close')
	}

	function closeKey(event: KeyboardEvent) {
		if (event.key == 'Escape') {
			dispatcher('close')
		}
	}
</script>

<svelte:window on:keyup={closeKey} />

<div
	class="fixed top-0 left-0 right-0 bottom-0 w-screen h-screen bg-slate-700 bg-opacity-80 p-4"
	style="z-index: 999"
	on:click={close}
	on:keyup={closeKey}
>
	{#if type == 'photo' || type == 'gif'}
		<img class="w-auto h-full mx-auto" src={url} alt="Original" />
	{:else if type == 'video'}
		<!-- svelte-ignore a11y-media-has-caption -->
		<video controls autoplay class="object-contain h-full mx-auto">
			<source src={url} type="video/mp4" />
		</video>
	{/if}
</div>
