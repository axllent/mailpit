<script>
import CommonMixins from '../mixins/CommonMixins'
import { mailbox } from '../stores/mailbox'
import { pagination } from '../stores/pagination'

export default {
	mixins: [CommonMixins],

	emits: ['loadMessages'],

	data() {
		return {
			mailbox,
			pagination,
		}
	},

	methods: {
		// test whether a tag is currently being searched for (in the URL)
		inSearch: function (tag) {
			const urlParams = new URLSearchParams(window.location.search)
			const query = urlParams.get('q')
			if (!query) {
				return false
			}

			let re = new RegExp(`(^|\\s)tag:"?${tag}"?($|\\s)`, 'i')
			return query.match(re)
		},

		// toggle a tag search in the search URL, add or remove it accordingly
		toggleTag: function (e, tag) {
			e.preventDefault()

			const urlParams = new URLSearchParams(window.location.search)
			let query = urlParams.get('q') ? urlParams.get('q') : ''

			let re = new RegExp(`(^|\\s)((-|\\!)?tag:"?${tag}"?)($|\\s)`, 'i')

			if (query.match(re)) {
				// remove is exists
				query = query.replace(re, '$1$4')
			} else {
				// add to query
				if (tag.match(/ /)) {
					tag = `"${tag}"`
				}
				query = query + " tag:" + tag
			}

			query = query.trim()

			if (query == '') {
				this.$router.push('/')
			} else {
				const params = new URLSearchParams({
					q: query,
					start: pagination.start.toString(),
					limit: pagination.limit.toString(),
				})
				this.$router.push('/search?' + params.toString())
			}
		},

		toTagUrl(t) {
			if (t.match(/ /)) {
				t = `"${t}"`
			}
			const p = {
				q: 'tag:' + t
			}
			if (pagination.limit != pagination.defaultLimit) {
				p.limit = pagination.limit.toString()
			}
			const params = new URLSearchParams(p)
			return '/search?' + params.toString()
		},
	}
}
</script>

<template>
	<template v-if="mailbox.tags && mailbox.tags.length">
		<div class="mt-4 text-muted">
			<button class="btn btn-sm dropdown-toggle ms-n1" data-bs-toggle="dropdown" aria-expanded="false">
				Tags
			</button>
			<ul class="dropdown-menu dropdown-menu-end">
				<li>
					<button class="dropdown-item" @click="mailbox.showTagColors = !mailbox.showTagColors">
						<template v-if="mailbox.showTagColors">Hide</template>
						<template v-else>Show</template>
						tag colors
					</button>
				</li>
			</ul>
		</div>
		<div class="list-group mt-1 mb-5 pb-3">
			<RouterLink v-for="tag in mailbox.tags" :to="toTagUrl(tag)" @click="hideNav"
				v-on:click="pagination.start = 0" v-on:click.ctrl="toggleTag($event, tag)"
				:style="mailbox.showTagColors ? { borderLeftColor: colorHash(tag), borderLeftWidth: '4px' } : ''"
				class="list-group-item list-group-item-action small px-2" :class="inSearch(tag) ? 'active' : ''">
				<i class="bi bi-tag-fill" v-if="inSearch(tag)"></i>
				<i class="bi bi-tag" v-else></i>
				{{ tag }}
			</RouterLink>
		</div>
	</template>
</template>
