<script lang="ts">
	import { gql, queryStore } from '@urql/svelte'

	import { page } from '$app/stores'
	import type { PageData } from './$types'
	import type { Query } from '$graph'
	import { PUBLIC_SERVER_URL } from '$env/static/public'
	import ProfileName from '$lib/components/ProfileName.svelte'

	export let data: PageData

	const profiles = queryStore<Query, {}>({
		client: data.client,
		query: gql`
			query ($source: Source!) {
				profiles(source: $source) {
					id
					name
					username
					avatar
				}
			}
		`,
		variables: {
			source: $page.params.source
		}
	})
</script>

{#if $profiles.fetching}
	<p>Loading...</p>
{:else if $profiles.error}
	<p>Oh no... {$profiles.error}</p>
{:else}
	<div class="grid w-full grid-cols-1 mx-auto md:grid-cols-2 lg:grid-cols-3">
		{#each $profiles?.data?.profiles || [] as profile}
			<div class="p-2">
				<ProfileName {profile} baseUrl={PUBLIC_SERVER_URL} />
			</div>
		{:else}
			<p>No posts</p>
		{/each}
	</div>
{/if}
