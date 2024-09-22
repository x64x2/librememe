<script lang="ts">
	import MediaCarousel from '$components/MediaCarousel.svelte'
	import ProfileName from '$components/ProfileName.svelte'
	import type { Post } from '$graph'
	import { format } from 'date-fns'

	export let baseUrl: string
	export let item: Post

	$: itemDate = new Date(item.date)
</script>

<div class="p-4 space-y-2">
	<!--{item.id}-->
	<div class="flex flex-row items-center justify-between mr-6">
		<ProfileName profile={item.author} {baseUrl} />
		<span class="flex-shrink-0 hidden text-sm text-gray-600 md:block">
			{format(itemDate, 'PPpp')}
		</span>
		<span class="flex-shrink-0 text-sm text-gray-600 md:hidden">
			{format(itemDate, 'P')}<br />
			{format(itemDate, 'p')}
		</span>
	</div>
	<p>{item.text}</p>
	{#if item.media && item.media.length > 0}
		<MediaCarousel {baseUrl} media={item.media} />
	{/if}
</div>
