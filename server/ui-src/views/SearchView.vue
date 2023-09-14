<script>
import CommonMixins from '../mixins/CommonMixins.js'
import MessagesMixins from '../mixins/MessagesMixins.js'

import AboutMailpit from "../components/AboutMailpit.vue"
import AjaxLoader from '../components/AjaxLoader.vue'
import ListMessages from "../components/ListMessages.vue"
import SearchActions from "../components/SearchActions.vue"
import MailboxTags from "../components/MailboxTags.vue"
import Pagination from "../components/Pagination.vue"
import SearchForm from "../components/SearchForm.vue"

import { mailbox } from "../stores/mailbox"
import { pagination } from "../stores/pagination"

export default {
	mixins: [CommonMixins, MessagesMixins],

	components: {
		AboutMailpit,
		AjaxLoader,
		ListMessages,
		SearchActions,
		MailboxTags,
		Pagination,
		SearchForm,
	},

	data() {
		return {
			mailbox,
			pagination,
		}
	},

	watch: {
		$route(to, from) {
			this.doSearch(true)
		}
	},

	mounted() {
		this.mailbox.searching = true
		this.doSearch(false)
	},

	methods: {
		doSearch: function (resetPagination) {
			const urlParams = new URLSearchParams(window.location.search);
			let s = urlParams.get('q') ? urlParams.get('q') : '';

			if (s == '') {
				this.$router.push('/')
				return
			}

			if (resetPagination) {
				pagination.start = 0
			}

			this.apiURI = this.$router.resolve(`/api/v1/search`).href + '?query=' + encodeURIComponent(s)
			this.loadMessages()
		}
	}
}
</script>

<template>
	<div class="navbar navbar-expand-lg navbar-dark row flex-shrink-0 bg-primary text-white">
		<div class="col-lg-2 col-md-3 d-none d-md-block">
			<RouterLink to="/" class="navbar-brand text-white">
				<img :src="baseURL + 'mailpit.svg'" alt=" Mailpit">
				<span class="ms-2">Mailpit</span>
			</RouterLink>
		</div>
		<div class="col col-md-9 col-lg-5">
			<SearchForm />
		</div>
		<div class="col-12 col-lg-5 text-end mt-2 mt-lg-0">
			<Pagination @loadMessages="loadMessages" :total="mailbox.count" />
		</div>
	</div>

	<div class="row flex-fill" style="min-height:0">
		<div class="d-none d-md-block col-lg-2 col-md-3 mh-100 position-relative"
			style="overflow-y: auto; overflow-x: hidden;">
			<SearchActions @loadMessages="loadMessages" />
			<MailboxTags />
			<AboutMailpit />
		</div>

		<div class="col-lg-10 col-md-9 mh-100 ps-0 ps-md-2 pe-0">
			<div class="mh-100" style="overflow-y: auto;" id="message-page">
				<ListMessages />
			</div>
		</div>
	</div>

	<AjaxLoader :loading="loading" />
</template>
