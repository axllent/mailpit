<script>
import AboutMailpit from '../components/AboutMailpit.vue'
import AjaxLoader from '../components/AjaxLoader.vue'
import CommonMixins from '../mixins/CommonMixins'
import ListMessages from '../components/ListMessages.vue'
import MessagesMixins from '../mixins/MessagesMixins'
import NavMailbox from '../components/NavMailbox.vue'
import NavTags from '../components/NavTags.vue'
import Pagination from '../components/Pagination.vue'
import SearchForm from '../components/SearchForm.vue'
import { mailbox } from '../stores/mailbox'
import { pagination } from "../stores/pagination";

export default {
	mixins: [CommonMixins, MessagesMixins],

	components: {
		AboutMailpit,
		AjaxLoader,
		ListMessages,
		NavMailbox,
		NavTags,
		Pagination,
		SearchForm,
	},

	data() {
		return {
			mailbox,
		}
	},

	watch: {
		$route(to, from) {
			this.loadMailbox()
		}
	},

	mounted() {
		mailbox.searching = false
		this.apiURI = this.resolve(`/api/v1/messages`)
		this.loadMailbox()
	},

	methods: {
		loadMailbox: function () {
			const paginationParams = this.getPaginationParams()
			if (paginationParams?.start) {
				pagination.start = paginationParams.start
			} else {
				pagination.start = 0
			}
			if (paginationParams?.limit) {
				pagination.limit = paginationParams.limit
			}

			this.loadMessages()
		}
	}
}
</script>

<template>
	<div class="navbar navbar-expand-lg navbar-dark row flex-shrink-0 bg-primary text-white">
		<div class="col-xl-2 col-md-3 col-auto pe-0">
			<RouterLink to="/" class="navbar-brand text-white me-0" @click="reloadMailbox">
				<img :src="resolve('/mailpit.svg')" alt="Mailpit">
				<span class="ms-2 d-none d-sm-inline">Mailpit</span>
			</RouterLink>
		</div>
		<div class="col col-md-4k col-lg-5 col-xl-6">
			<SearchForm />
		</div>
		<div class="col-12 col-md-auto col-lg-4 col-xl-4 text-end mt-2 mt-md-0">
			<div class="float-start d-md-none">
				<button class="btn btn-outline-light me-2" type="button" data-bs-toggle="offcanvas"
					data-bs-target="#offcanvas" aria-controls="offcanvas">
					<i class="bi bi-list"></i>
				</button>
			</div>
			<Pagination :total="mailbox.total" />
		</div>
	</div>

	<div class="offcanvas-md offcanvas-start d-md-none" data-bs-scroll="true" tabindex="-1" id="offcanvas"
		aria-labelledby="offcanvasLabel">
		<div class="offcanvas-header">
			<h5 class="offcanvas-title" id="offcanvasLabel">Mailpit</h5>
			<button type="button" class="btn-close" data-bs-dismiss="offcanvas" data-bs-target="#offcanvas"
				aria-label="Close"></button>
		</div>
		<div class="offcanvas-body">
			<NavMailbox @loadMessages="loadMessages" />
			<NavTags />
			<AboutMailpit />
		</div>
	</div>

	<div class="row flex-fill" style="min-height:0">
		<div class="d-none d-md-block col-xl-2 col-md-3 mh-100 position-relative"
			style="overflow-y: auto; overflow-x: hidden;">
			<NavMailbox @loadMessages="loadMessages" />
			<NavTags />
			<AboutMailpit />
		</div>

		<div class="col-xl-10 col-md-9 mh-100 ps-0 ps-md-2 pe-0">
			<div class="mh-100" style="overflow-y: auto;" id="message-page">
				<ListMessages :loading-messages="loading" />
			</div>
		</div>
	</div>

	<NavMailbox @loadMessages="loadMessages" modals />
	<AboutMailpit modals />
	<AjaxLoader :loading="loading" />
</template>
