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

	// global event bus to handle message status changes
	inject: ["eventBus"],

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
			delayedRefresh: false,
			paginationDelayed: false, // for delayed pagination URL changes
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

		// subscribe to events
		this.eventBus.on("new", this.handleWSNew)
		this.eventBus.on("update", this.handleWSUpdate)
		this.eventBus.on("delete", this.handleWSDelete)
		this.eventBus.on("truncate", this.handleWSTruncate)
	},

	unmounted() {
		// unsubscribe from events
		this.eventBus.off("new", this.handleWSNew)
		this.eventBus.off("update", this.handleWSUpdate)
		this.eventBus.off("delete", this.handleWSDelete)
		this.eventBus.off("truncate", this.handleWSTruncate)
	},

	methods: {
		loadMailbox() {
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
		},

		// This will only update the pagination offset at a maximum of 2x per second
		// when viewing the inbox on > page 1, while receiving an influx of new messages.
		delayedPaginationUpdate() {
			if (this.paginationDelayed) {
				return
			}

			this.paginationDelayed = true

			window.setTimeout(() => {
				const path = this.$route.path
				const p = {
					...this.$route.query
				}
				if (pagination.start > 0) {
					p.start = pagination.start.toString()
				} else {
					delete p.start
				}
				if (pagination.limit != pagination.defaultLimit) {
					p.limit = pagination.limit.toString()
				} else {
					delete p.limit
				}

				mailbox.autoPaginating = false // prevent reload of messages when URL changes
				const params = new URLSearchParams(p)
				this.$router.replace(path + '?' + params.toString())

				this.paginationDelayed = false
			}, 500)
		},

		// handler for websocket new messages
		handleWSNew(data) {
			if (pagination.start < 1) {
				// push results directly into first page
				mailbox.messages.unshift(data)
				if (mailbox.messages.length > pagination.limit) {
					mailbox.messages.pop()
				}
			} else {
				// update pagination offset
				pagination.start++
				// prevent "Too many calls to Location or History APIs within a short time frame"
				this.delayedPaginationUpdate()
			}
		},

		// handler for websocket message updates
		handleWSUpdate(data) {
			for (let x = 0; x < this.mailbox.messages.length; x++) {
				if (this.mailbox.messages[x].ID == data.ID) {
					// update message
					this.mailbox.messages[x] = { ...this.mailbox.messages[x], ...data }
					return
				}
			}
		},

		// handler for websocket message deletion
		handleWSDelete(data) {
			let removed = 0;
			for (let x = 0; x < this.mailbox.messages.length; x++) {
				if (this.mailbox.messages[x].ID == data.ID) {
					// remove message from the list
					this.mailbox.messages.splice(x, 1)
					removed++
					continue
				}
			}

			if (!removed || this.delayedRefresh) {
				// nothing changed on this screen, or a refresh is queued,
				// don't refresh
				return
			}

			// delayedRefresh prevents unnecessary reloads when multiple messages are deleted
			this.delayedRefresh = true

			window.setTimeout(() => {
				this.delayedRefresh = false
				this.loadMessages()
			}, 500)
		},

		// handler for websocket message truncation
		handleWSTruncate() {
			// all messages gone, reload
			this.loadMessages()
		},
	}
}
</script>

<template>
	<div class="navbar navbar-expand-lg navbar-dark row flex-shrink-0 bg-primary text-white d-print-none">
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
		<div class="offcanvas-body pb-0">
			<div class="d-flex flex-column h-100">
				<div class="flex-grow-1 overflow-y-auto">

					<NavMailbox @loadMessages="loadMessages" />
					<NavTags />
				</div>
				<AboutMailpit />
			</div>
		</div>
	</div>

	<div class="row flex-fill" style="min-height:0">
		<div class="d-none d-md-flex h-100 col-xl-2 col-md-3 flex-column">
			<div class="flex-grow-1 overflow-y-auto">
				<NavMailbox @loadMessages="loadMessages" />
				<NavTags />
			</div>
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
