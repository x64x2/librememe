import { sveltekit } from '@sveltejs/kit/vite'
import type { UserConfig } from 'vite'
import path from 'path'

const config: UserConfig = {
	plugins: [sveltekit()],
	optimizeDeps: {
		exclude: ['@urql/svelte']
	},
	server: {
		port: 8090
	},
	resolve: {
		alias: {
			$components: path.resolve('./src/lib/components'),
			$graph: './src/graph'
		}
	}
}

export default config
