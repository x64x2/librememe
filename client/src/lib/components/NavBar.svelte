<script lang="ts">
	import { page } from '$app/stores'

	import NavLink from '$components/NavLink.svelte'
</script>

<nav class="w-full max-w-2xl py-2 mx-auto">
	<div class="flex flex-row justify-center space-x-5 align-middle">
		<NavLink title="Feed" link="/{$page.params.source}/feed" active={$page.route.id == '/[source]/feed'} />
		<NavLink title="Profiles" link="/{$page.params.source}/profiles" active={$page.route.id == '/[source]/profiles'} />
	</div>
</nav>
