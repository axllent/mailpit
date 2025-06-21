<script>
import About from "../components/AppAbout.vue";
import CommonMixins from "../mixins/CommonMixins";

export default {
	components: {
		About,
	},

	mixins: [CommonMixins],
};
</script>

<template>
	<div class="h-100 bg-primary d-flex align-items-center justify-content-center my-2 text-white">
		<div class="d-block text-center">
			<RouterLink to="/" class="text-white">
				<img :src="resolve('/mailpit.svg')" alt="Mailpit" style="max-width: 80%; width: 100px" />
				<p class="h2 my-3">Page not found</p>

				<p>Click here to continue</p>
			</RouterLink>
		</div>

		<div class="d-none">
			<About />
		</div>
	</div>
</template>
