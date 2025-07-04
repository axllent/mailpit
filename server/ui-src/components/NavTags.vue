<script>
import CommonMixins from "../mixins/CommonMixins";
import { mailbox } from "../stores/mailbox";
import { pagination } from "../stores/pagination";

export default {
	mixins: [CommonMixins],

	data() {
		return {
			mailbox,
			pagination,
		};
	},

	methods: {
		// test whether a tag is currently being searched for (in the URL)
		inSearch(tag) {
			const urlParams = new URLSearchParams(window.location.search);
			const query = urlParams.get("q");
			if (!query) {
				return false;
			}

			const re = new RegExp(`(^|\\s)tag:("${tag}"|${tag}\\b)`, "i");
			return query.match(re);
		},

		// toggle a tag search in the search URL, add or remove it accordingly
		toggleTag(e, tag) {
			e.preventDefault();

			const urlParams = new URLSearchParams(window.location.search);
			let query = urlParams.get("q") ? urlParams.get("q") : "";

			const re = new RegExp(`(^|\\s)((-|\\!)?tag:"?${tag}"?)($|\\s)`, "i");

			if (query.match(re)) {
				// remove is exists
				query = query.replace(re, "$1$4");
			} else {
				// add to query
				if (tag.match(/ /)) {
					tag = `"${tag}"`;
				}
				query = query + " tag:" + tag;
			}

			query = query.trim();

			if (query === "") {
				this.$router.push("/");
			} else {
				const params = new URLSearchParams({
					q: query,
					start: pagination.start.toString(),
					limit: pagination.limit.toString(),
				});
				this.$router.push("/search?" + params.toString());
			}
		},

		toTagUrl(t) {
			if (t.match(/ /)) {
				t = `"${t}"`;
			}
			const p = {
				q: "tag:" + t,
			};
			if (pagination.limit !== pagination.defaultLimit) {
				p.limit = pagination.limit.toString();
			}
			const params = new URLSearchParams(p);
			return "/search?" + params.toString();
		},
	},
};
</script>

<template>
	<template v-if="mailbox.tags && mailbox.tags.length">
		<div class="mt-4 text-muted">
			<button class="btn btn-sm dropdown-toggle" data-bs-toggle="dropdown" aria-expanded="false">Tags</button>
			<ul class="dropdown-menu dropdown-menu-end">
				<li>
					<button class="dropdown-item" data-bs-toggle="modal" data-bs-target="#EditTagsModal">
						Edit tags
					</button>
				</li>
				<li>
					<button class="dropdown-item" @click="mailbox.showTagColors = !mailbox.showTagColors">
						<template v-if="mailbox.showTagColors">Hide</template>
						<template v-else>Show</template>
						tag colors
					</button>
				</li>
			</ul>
		</div>
		<div class="list-group mt-1 mb-2">
			<RouterLink
				v-for="tag in mailbox.tags"
				:key="tag"
				:to="toTagUrl(tag)"
				:style="mailbox.showTagColors ? { borderLeftColor: colorHash(tag), borderLeftWidth: '4px' } : ''"
				class="list-group-item list-group-item-action small px-2"
				:class="inSearch(tag) ? 'active' : ''"
				@click.exact="hideNav"
				@click="pagination.start = 0"
				@click.meta="toggleTag($event, tag)"
				@click.ctrl="toggleTag($event, tag)"
			>
				<i v-if="inSearch(tag)" class="bi bi-tag-fill"></i>
				<i v-else class="bi bi-tag"></i>
				{{ tag }}
			</RouterLink>
		</div>
	</template>
</template>
