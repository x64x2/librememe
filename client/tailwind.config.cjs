/** @type {import('tailwindcss').Config} */
module.exports = {
	content: ['./src/**/*.{html,js,svelte,ts}'],
	theme: {
		extend: {
			spacing: {
				128: '32rem',
				'screen-75': '75vh'
			},
			minHeight: {
				60: '15rem',
				64: '16rem',
				72: '18rem',
				80: '20rem',
				96: '24rem'
			}
		}
	},
	plugins: []
}
