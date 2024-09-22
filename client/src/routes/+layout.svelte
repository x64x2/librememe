<script lang="ts">
	import '../app.css'
</script>

<svelte:head>
	<title>myfans</title>
</svelte:head>

<main class="w-full py-2 mx-auto md:w-2/3">
	<slot />
</main>

<footer class="w-full py-2 mx-auto md:w-2/3">
	<p class="text-center">
		Powered by <a
			class="text-gray-800 underline hover:text-gray-700"
			href="https://codeberg.org/biggestfan24/myfans">myfans</a
		>
	</p>
</footer>
