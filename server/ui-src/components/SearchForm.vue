<script>
import CommonMixins from '../mixins/CommonMixins.js'

export default {
	mixins: [CommonMixins],

	data() {
		return {
			search: ''
		}
	},

	mounted() {
		this.searchFromURL()
	},

	watch: {
		$route() {
			this.searchFromURL()
		}
	},

	methods: {
		searchFromURL: function () {
			const urlParams = new URLSearchParams(window.location.search);
			this.search = urlParams.get('q') ? urlParams.get('q') : '';
		},

		doSearch: function (e) {
			// let u = this.$router.resolve(`/search`).href;
			if (this.search == '') {
				this.$router.push('/')
			} else {
				this.$router.push('/search?q=' + encodeURIComponent(this.search))
			}

			e.preventDefault()
		},

		resetSearch: function () {
			this.search = ''
			this.$router.push('/')
		}
	}
}
</script>

<template>
	<form v-on:submit="doSearch">
		<div class="input-group">
			<RouterLink to="/" class="navbar-brand d-md-none">
				<img :src="baseURL + 'mailpit.svg'" alt="Mailpit">
			</RouterLink>
			<div class="ms-md-2 d-flex border bg-body rounded-start flex-fill position-relative">
				<input type="text" class="form-control border-0" aria-label="Search" v-model.trim="search"
					placeholder="Search mailbox">
				<span class="btn btn-link position-absolute end-0 text-muted" v-if="search != ''"
					v-on:click="resetSearch"><i class="bi bi-x-circle"></i></span>
			</div>
			<button class="btn btn-outline-secondary" type="submit">
				<i class="bi bi-search"></i>
			</button>
		</div>
	</form>
</template>
