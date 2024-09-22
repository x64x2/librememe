<script lang="ts">
	import SliderNext from '$components/SliderNext.svelte'
	import SliderPrevious from '$components/SliderPrevious.svelte'
	import type { Media } from '$graph'
	import PlayButton from './PlayButton.svelte'
	import MediaLocked from './MediaLocked.svelte'
	import Lightbox from './Lightbox.svelte'

	export let baseUrl: string
	export let media: Media[]

	let active = 0
	let lightbox = false

	function next() {
		active++
		if (active >= media.length) {
			active = 0
		}
	}

	function prev() {
		active--
		if (active < 0) {
			active = media.length - 1
		}
	}

	function setActive(i: number) {
		active = i
	}

	function mediaPreview(media: Media): string {
		return media.preview || media.location || ''
	}
</script>

<div class="relative w-full p-1 rounded-lg bg-slate-700">
	{#if lightbox && media[active].location}
		<Lightbox
			url={baseUrl + '/file/' + media[active].location}
			type={media[active].type}
			on:close={() => (lightbox = false)}
		/>
	{/if}
	{#if !media[active].visible}
		<MediaLocked />
	{:else if (media[active].type == 'video' || media[active].type == 'gif') && media[active].location != ''}
		<PlayButton on:click={() => (lightbox = true)} />
	{/if}
	<div class="flex flex-row items-center justify-center w-full h-screen-75 min-h-80">
		{#if mediaPreview(media[active])}
			<img
				src={baseUrl + '/file/' + mediaPreview(media[active])}
				class="object-contain h-full"
				alt="Preview"
				on:click={() => (lightbox = true)}
			/>
		{/if}
	</div>
	<div class="flex items-center justify-center mt-2 mb-1 space-x-2">
		{#each media as _, i}
			<button
				type="button"
				class={i == active
					? 'w-2 h-2 bg-white rounded-full dark:bg-gray-800'
					: 'w-2 h-2 rounded-full bg-white/50 dark:bg-gray-800/50 hover:bg-white dark:hover:bg-gray-800'}
				aria-current={i == active ? 'true' : 'false'}
				aria-label="Slide to {i + 1}"
				on:click={() => setActive(i)}
			/>
		{/each}
	</div>
	{#if media.length > 1}
		<SliderPrevious {prev} />
		<SliderNext {next} />
	{/if}
</div>
