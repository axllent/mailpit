<script>
import CommonMixins from "../mixins/CommonMixins";
import { pagination } from "../stores/pagination";

export default {
	mixins: [CommonMixins],

	emits: ["loadMessages"],

	data() {
		return {
			search: "",
		};
	},

	watch: {
		$route() {
			this.searchFromURL();
		},
	},

	mounted() {
		this.searchFromURL();
	},

	methods: {
		searchFromURL() {
			const urlParams = new URLSearchParams(window.location.search);
			this.search = urlParams.get("q") ? urlParams.get("q") : "";
		},

		doSearch(e) {
			pagination.start = 0;
			if (this.search === "") {
				this.$router.push("/");
			} else {
				const urlParams = new URLSearchParams(window.location.search);
				const curr = urlParams.get("q");
				if (curr && curr === this.search) {
					pagination.start = 0;
					this.$emit("loadMessages");
				}
				const p = {
					q: this.search,
				};
				if (pagination.start > 0) {
					p.start = pagination.start.toString();
				}
				if (pagination.limit !== pagination.defaultLimit) {
					p.limit = pagination.limit.toString();
				}

				const params = new URLSearchParams(p);
				this.$router.push("/search?" + params.toString());
			}

			e.preventDefault();
		},

		resetSearch() {
			this.search = "";
			this.$router.push("/");
		},
	},
};
</script>

<template>
	<form @submit="doSearch">
		<div class="input-group flex-nowrap">
			<div class="ms-md-2 d-flex border bg-body rounded-start flex-fill position-relative">
				<input
					v-model.trim="search"
					type="text"
					class="form-control border-0"
					aria-label="Search"
					placeholder="Search mailbox"
				/>
				<span v-if="search != ''" class="btn btn-link position-absolute end-0 text-muted" @click="resetSearch"
					><i class="bi bi-x-circle"></i
				></span>
			</div>
			<button class="btn btn-outline-secondary" type="submit">
				<i class="bi bi-search"></i>
			</button>
		</div>
	</form>
</template>
