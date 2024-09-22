<script lang="ts">
	export let link: string
	export let title: string
	export let active: boolean = false

	$: classes = active ? 'font-bold text-md underline' : 'font-bold text-md text-gray-600'
</script>

<a href={link} class={classes}>{title}</a>
