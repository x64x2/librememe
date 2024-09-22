<script lang="ts">
	import { gql } from '@urql/svelte'
	import { onMount } from 'svelte'
	import { formatISO as dateFnsFormatISO } from 'date-fns'

	import PostViewer from '$components/PostViewer.svelte'
	import FeedControls from '$components/FeedControls.svelte'
	import type { Query, Post, Profile, ProfilePostsArgs, QueryProfileArgs } from '$graph'
	import type { PageData } from './$types'
	import { page } from '$app/stores'
	import { goto } from '$app/navigation'
	import { PUBLIC_SERVER_URL } from '$env/static/public'

	export let data: PageData

	let onlyWithMedia = true
	let onlyVisible = true
	let before: Date | undefined
	let after: Date | undefined
	let sort: string | undefined

	type queryArgs = ProfilePostsArgs & QueryProfileArgs

	const profileQuery = gql`
		query (
			$source: Source!
			$username: String
		) {
			profile(
				source: $source,
				username: $username
			) {
				id
				sourceId
				username
				name
				avatar
				header
			}
		}
	`

	const feedQuery = gql`
		query (
			$source: Source
			$onlyWithMedia: Boolean
			$onlyVisible: Boolean
			$username: String
			$before: Time
			$after: Time
			$sort: Sort
		) {
			profile(
				source: $source,
				username: $username
			) {
				posts(
					onlyWithMedia: $onlyWithMedia,
					onlyVisible: $onlyVisible,
					before: $before,
					after: $after,
					sort: $sort
				) {
					id
					sourceId
					author {
						id
						sourceId
					}
					date
					text
					media {
						type
						location
						preview
						visible
					}
				}
			}
		}
	`

	let userProfiles: Record<string, Profile> = {}
	$: userProfilePromise = fetchUserProfile($page.params.user)
	function fetchUserProfile(user: string): Promise<any> {
		return data.client
			.query<Query, queryArgs>(profileQuery, {
				username: user,
				source: $page.params.source
			})
			.toPromise()
			.then((res) => {
				if (res?.data?.profile?.id) {
					userProfiles[res.data.profile.id] = res.data.profile
				}
			})
			.catch((err) => {
				console.error(
					'Error while requesting profile for user ' +
						(user || 'undefined') +
						': ' +
						err
				)
			})
	}

	let feed: Post[] = []
	let fetching = true
	let pageError: String | undefined
	let nextFunc: () => Promise<boolean>
	let hasNext = false

	async function invokeNext() {
		hasNext = await nextFunc()
	}

	async function newLoadFeed() {
		nextFunc = await loadFeed($page.params.user, onlyWithMedia, onlyVisible, before, after, sort)
		invokeNext()
	}

	$: onlyVisible || newLoadFeed()
	$: onlyWithMedia || newLoadFeed()

	function pickedDate(event: CustomEvent) {
		const url = $page.url
		if (before) {
			before.setHours(23)
			before.setMinutes(59)
			before.setSeconds(59)
			url.searchParams.set('before', dateFnsFormatISO(before))
		} else {
			url.searchParams.delete('before')
		}
		goto(url.toString())
	}

	onMount(() => {
		page.subscribe((p) => {
			const params = p.url.searchParams
			before = params.get('before') ? new Date(params.get('before')!) || undefined : undefined
			after = params.get('after') ? new Date(params.get('after')!) || undefined : undefined

			switch ((params.get('sort') || '').toLowerCase()) {
				case 'asc':
				case 'ascending':
				case '1':
				case '+1':
					sort = 'asc'
					break
				case 'desc':
				case 'descending':
				case 'des':
				case '-1':
					sort = 'desc'
					break
				default:
					sort = undefined
					break
			}

			newLoadFeed()
		})
	})

	async function loadFeed(
		username: string,
		onlyWithMedia: boolean,
		onlyVisible: boolean,
		before?: Date,
		after?: Date,
		sort?: string
	): Promise<() => Promise<boolean>> {
		feed = []
		fetching = true
		pageError = undefined

		await userProfilePromise

		let cursor = before

		return (): Promise<boolean> => {
			return data.client
				.query<Query, queryArgs>(feedQuery, {
					onlyWithMedia,
					onlyVisible,
					username: username,
					source: $page.params.source,
					//sort,
					before: cursor,
					after
				})
				.toPromise()
				.then((res) => {
					fetching = false

					if (res?.error) {
						pageError = res.error.toString()
						return false
					}
					if (!res?.data) {
						pageError = 'Response is empty'
						return false
					}

					if (res.data?.profile?.posts && res.data.profile.posts.length > 0) {
						feed = [...feed, ...res.data.profile.posts]
						const last = res.data.profile.posts[res.data.profile.posts.length - 1]
						if (last.date) {
							if (typeof last.date == 'object' && last.date instanceof Date) {
								cursor = last.date
							} else {
								cursor = new Date(last.date + '')
							}
							return true
						}
						return false
					}

					return false
				})
				.catch((err) => {
					fetching = false
					pageError = 'Error requesting data: ' + err
					return false
				})
		}
	}

	function PostWithAuthor(item: Post): Post {
		if (item.author?.id && userProfiles[item.author.id]) {
			item.author = userProfiles[item.author.id]
		}
		return item
	}
</script>

<FeedControls bind:onlyVisible bind:date={before} on:pickedDate={pickedDate} />

{#if fetching}
	<p>Loading...</p>
{:else if pageError}
	<p>Oh no... {pageError}</p>
{:else}
	<div class="grid grid-cols-1 divide-y">
		{#if feed && feed.length > 0}
			{#each feed as item}
				<PostViewer item={PostWithAuthor(item)} baseUrl={PUBLIC_SERVER_URL} />
			{/each}
			{#if hasNext}
				<button on:click={invokeNext}>More</button>
			{/if}
		{:else}
			<p>No posts</p>
		{/if}
	</div>
{/if}
