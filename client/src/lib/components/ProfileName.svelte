<script lang="ts">
	import type { Profile } from '$graph'

	export let baseUrl: string
	export let profile: Profile
</script>

<a
	href="profile/{profile.username}"
	class="flex flex-row items-center w-full overflow-hidden justify-left"
>
	{#if profile.avatar}
		<img
			src={baseUrl + '/file/' + profile.avatar}
			alt={profile.username}
			class="flex-none mx-4 rounded-full w-14 h-14"
		/>
	{:else}
		<span
			class="mx-4 text-4xl font-semibold text-center text-gray-700 align-middle bg-gray-100 border border-gray-700 rounded-full w-14 h-14"
			>{(profile.username || ' ').charAt(0)}</span
		>
	{/if}
	<p class="flex flex-col mb-2">
		<span class="font-bold">
			{#if profile.name}
				{profile.name}
			{/if}
		</span>
		<span class="text-sm text-gray-600">@{profile.username}</span>
	</p>
</a>
