import { createClient, cacheExchange, fetchExchange } from '@urql/svelte'
import type { LayoutLoad } from './$types'
import { PUBLIC_SERVER_URL } from '$env/static/public'

export const load: LayoutLoad = (event) => {
	const client = createClient({
		url: PUBLIC_SERVER_URL + '/query',
		fetch: fetch,
		exchanges: [cacheExchange, fetchExchange]
	})

	return {
		client
	}
}

export const ssr = false
export const prerender = false
