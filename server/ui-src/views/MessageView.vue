<script>
import AboutMailpit from "../components/AppAbout.vue";
import AjaxLoader from "../components/AjaxLoader.vue";
import CommonMixins from "../mixins/CommonMixins";
import Message from "../components/message/MessageItem.vue";
import Release from "../components/message/MessageRelease.vue";
import Screenshot from "../components/message/MessageScreenshot.vue";
import { mailbox } from "../stores/mailbox";
import { pagination } from "../stores/pagination";
import dayjs from "dayjs";

export default {
	components: {
		AboutMailpit,
		AjaxLoader,
		Message,
		Screenshot,
		Release,
	},

	mixins: [CommonMixins],

	// global event bus to handle message status changes
	inject: ["eventBus"],

	data() {
		return {
			mailbox,
			pagination,
			message: false,
			errorMessage: false,
			apiSideNavURI: false,
			apiSideNavParams: URLSearchParams,
			apiIsMore: true,
			messagesList: [],
			liveLoaded: 0, // the number new messages prepended tp messageList
			scrollLoading: false,
			canLoadMore: true,
		};
	},

	computed: {
		// get current message read status
		isRead() {
			const l = this.messagesList.length;
			if (!this.message || !l) {
				return true;
			}

			for (let x = 0; x < l; x++) {
				if (this.messagesList[x].ID === this.message.ID) {
					return this.messagesList[x].Read;
				}
			}

			return true;
		},

		// get the previous message ID
		previousID() {
			const l = this.messagesList.length;
			if (!this.message || !l) {
				return false;
			}

			let id = false;
			for (let x = 0; x < l; x++) {
				if (this.messagesList[x].ID === this.message.ID) {
					return id;
				}
				id = this.messagesList[x].ID;
			}

			return false;
		},

		// get the next message ID
		nextID() {
			const l = this.messagesList.length;
			if (!this.message || !l) {
				return false;
			}

			let id = false;
			for (let x = l - 1; x > 0; x--) {
				if (this.messagesList[x].ID === this.message.ID) {
					return id;
				}
				id = this.messagesList[x].ID;
			}

			return id;
		},
	},

	watch: {
		$route() {
			this.loadMessage();
		},
	},

	created() {
		const relativeTime = require("dayjs/plugin/relativeTime");
		dayjs.extend(relativeTime);

		this.initLoadMoreAPIParams();
	},

	mounted() {
		this.loadMessage();

		this.messagesList = JSON.parse(JSON.stringify(this.mailbox.messages));
		if (!this.messagesList.length) {
			this.loadMore();
		}

		this.refreshUI();

		// subscribe to events
		this.eventBus.on("new", this.handleWSNew);
		this.eventBus.on("update", this.handleWSUpdate);
		this.eventBus.on("delete", this.handleWSDelete);
		this.eventBus.on("truncate", this.handleWSTruncate);
	},

	unmounted() {
		// unsubscribe from events
		this.eventBus.off("new", this.handleWSNew);
		this.eventBus.off("update", this.handleWSUpdate);
		this.eventBus.off("delete", this.handleWSDelete);
		this.eventBus.off("truncate", this.handleWSTruncate);
	},

	methods: {
		loadMessage() {
			this.message = false;
			const uri = this.resolve("/api/v1/message/" + this.$route.params.id);
			this.get(
				uri,
				false,
				(response) => {
					this.errorMessage = false;
					const d = response.data;

					// update read status in case websockets is not working
					this.handleWSUpdate({ ID: d.ID, Read: true });

					// replace inline images embedded as inline attachments
					if (d.HTML && d.Inline) {
						for (const i in d.Inline) {
							const a = d.Inline[i];
							if (a.ContentID !== "") {
								d.HTML = d.HTML.replace(
									new RegExp("(=[\"']?)(cid:" + a.ContentID + ")([\"|'|\\s|\\/|>|;])", "g"),
									"$1" + this.resolve("/api/v1/message/" + d.ID + "/part/" + a.PartID) + "$3",
								);
							}
							if (a.FileName.match(/^[a-zA-Z0-9_\-.]+$/)) {
								// some old email clients use the filename
								d.HTML = d.HTML.replace(
									new RegExp("(=[\"']?)(" + a.FileName + ")([\"|'|\\s|\\/|>|;])", "g"),
									"$1" + this.resolve("/api/v1/message/" + d.ID + "/part/" + a.PartID) + "$3",
								);
							}
						}
					}

					// replace inline images embedded as regular attachments
					if (d.HTML && d.Attachments) {
						for (const i in d.Attachments) {
							const a = d.Attachments[i];
							if (a.ContentID !== "") {
								d.HTML = d.HTML.replace(
									new RegExp("(=[\"']?)(cid:" + a.ContentID + ")([\"|'|\\s|\\/|>|;])", "g"),
									"$1" + this.resolve("/api/v1/message/" + d.ID + "/part/" + a.PartID) + "$3",
								);
							}
							if (a.FileName.match(/^[a-zA-Z0-9_\-.]+$/)) {
								// some old email clients use the filename
								d.HTML = d.HTML.replace(
									new RegExp("(=[\"']?)(" + a.FileName + ")([\"|'|\\s|\\/|>|;])", "g"),
									"$1" + this.resolve("/api/v1/message/" + d.ID + "/part/" + a.PartID) + "$3",
								);
							}
						}
					}

					this.message = d;

					this.$nextTick(() => {
						this.scrollSidebarToCurrent();
					});
				},
				(error) => {
					this.errorMessage = true;
					if (error.response && error.response.data) {
						if (error.response.data.Error) {
							this.errorMessage = error.response.data.Error;
						} else {
							this.errorMessage = error.response.data;
						}
					} else if (error.request) {
						// The request was made but no response was received
						this.errorMessage = "Error sending data to the server. Please refresh the page.";
					} else {
						// Something happened in setting up the request that triggered an Error
						this.errorMessage = error.message;
					}
				},
			);
		},

		// UI refresh ticker to adjust relative times
		refreshUI() {
			window.setTimeout(() => {
				this.$forceUpdate();
				this.refreshUI();
			}, 30000);
		},

		// handler for websocket new messages
		handleWSNew(data) {
			// do not add when searching or >= 100 new messages have been received
			if (this.mailbox.searching || this.liveLoaded >= 100) {
				return;
			}

			this.liveLoaded++;
			this.messagesList.unshift(data);
		},

		// handler for websocket message updates
		handleWSUpdate(data) {
			for (let x = 0; x < this.messagesList.length; x++) {
				if (this.messagesList[x].ID === data.ID) {
					// update message
					this.messagesList[x] = { ...this.messagesList[x], ...data };
					return;
				}
			}
		},

		// handler for websocket message deletion
		handleWSDelete(data) {
			for (let x = 0; x < this.messagesList.length; x++) {
				if (this.messagesList[x].ID === data.ID) {
					// remove message from the list
					this.messagesList.splice(x, 1);
					return;
				}
			}
		},

		// handler for websocket message truncation
		handleWSTruncate() {
			// all messages gone, go to inbox
			this.$router.push("/");
		},

		// return whether the sidebar is visible
		sidebarVisible() {
			return this.$refs.MessageList.offsetParent !== null;
		},

		// scroll sidenav to current message if found
		scrollSidebarToCurrent() {
			const cont = document.getElementById("MessageList");
			if (!cont) {
				return;
			}
			const c = cont.querySelector(".router-link-active");
			if (c) {
				const outer = cont.getBoundingClientRect();
				const li = c.getBoundingClientRect();
				if (outer.top > li.top || outer.bottom < li.bottom) {
					c.scrollIntoView({
						behavior: "smooth",
						block: "center",
						inline: "nearest",
					});
				}
			}
		},

		scrollHandler(e) {
			if (!this.canLoadMore || this.scrollLoading) {
				return;
			}

			const { scrollTop, offsetHeight, scrollHeight } = e.target;
			if (scrollTop + offsetHeight + 150 >= scrollHeight) {
				this.loadMore();
			}
		},

		loadMore() {
			if (this.messagesList.length) {
				// get last created timestamp
				const oldest = this.messagesList[this.messagesList.length - 1].Created;
				// if set append `before=<ts>`
				this.apiSideNavParams.set("before", oldest);
			}

			this.scrollLoading = true;

			this.get(
				this.apiSideNavURI,
				this.apiSideNavParams,
				(response) => {
					if (response.data.messages.length) {
						this.messagesList.push(...response.data.messages);
					} else {
						this.canLoadMore = false;
					}
					this.$nextTick(() => {
						this.scrollLoading = false;
					});
				},
				null,
				true,
			);
		},

		initLoadMoreAPIParams() {
			let apiURI = this.resolve(`/api/v1/messages`);
			const p = {};

			if (mailbox.searching) {
				apiURI = this.resolve(`/api/v1/search`);
				p.query = mailbox.searching;
			}

			if (pagination.limit !== pagination.defaultLimit) {
				p.limit = pagination.limit.toString();
			}

			this.apiSideNavURI = apiURI;

			this.apiSideNavParams = new URLSearchParams(p);
		},

		getRelativeCreated(message) {
			const d = new Date(message.Created);
			return dayjs(d).fromNow();
		},

		getPrimaryEmailTo(message) {
			if (message.To && message.To.length > 0) {
				return message.To[0].Address;
			}

			return "[ Undisclosed recipients ]";
		},

		isActive(id) {
			return this.message.ID === id;
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

		downloadMessageBody(str, ext) {
			const dl = document.createElement("a");
			dl.href = "data:text/plain," + encodeURIComponent(str);
			dl.target = "_blank";
			dl.download = this.message.ID + "." + ext;
			dl.click();
		},

		screenshotMessageHTML() {
			this.$refs.ScreenshotRef.initScreenshot();
		},

		// toggle current message read status
		toggleRead() {
			if (!this.message) {
				return false;
			}
			const read = !this.isRead;

			const ids = [this.message.ID];
			const uri = this.resolve("/api/v1/messages");
			this.put(uri, { Read: read, IDs: ids }, () => {
				if (!this.sidebarVisible()) {
					return this.goBack();
				}

				// manually update read status in case websockets is not working
				this.handleWSUpdate({ ID: this.message.ID, Read: read });
			});
		},

		deleteMessage() {
			const ids = [this.message.ID];
			const uri = this.resolve("/api/v1/messages");
			// calculate next ID before deletion to prevent WS race
			const goToID = this.nextID ? this.nextID : this.previousID;

			this.delete(uri, { IDs: ids }, () => {
				if (!this.sidebarVisible()) {
					return this.goBack();
				}
				if (goToID) {
					return this.$router.push("/view/" + goToID);
				}

				return this.goBack();
			});
		},

		// return to mailbox or search based on origin
		goBack() {
			mailbox.lastMessage = this.$route.params.id;

			if (mailbox.searching) {
				const p = {
					q: mailbox.searching,
				};
				if (pagination.start > 0) {
					p.start = pagination.start.toString();
				}
				if (pagination.limit !== pagination.defaultLimit) {
					p.limit = pagination.limit.toString();
				}
				this.$router.push("/search?" + new URLSearchParams(p).toString());
			} else {
				const p = {};
				if (pagination.start > 0) {
					p.start = pagination.start.toString();
				}
				if (pagination.limit !== pagination.defaultLimit) {
					p.limit = pagination.limit.toString();
				}
				this.$router.push("/?" + new URLSearchParams(p).toString());
			}
		},

		reloadWindow() {
			location.reload();
		},

		initReleaseModal() {
			this.modal("ReleaseModal").show();
			window.setTimeout(() => {
				// delay to allow elements to load / focus
				this.$refs.ReleaseRef.initTags();
			}, 500);
		},
	},
};
</script>

<template>
	<div class="navbar navbar-expand-lg row flex-shrink-0 bg-primary text-white d-print-none" data-bs-theme="dark">
		<div class="d-none d-xl-block col-xl-3 col-auto pe-0">
			<RouterLink to="/" class="navbar-brand text-white me-0" @click="pagination.start = 0">
				<img :src="resolve('/mailpit.svg')" alt="Mailpit" />
				<span class="ms-2 d-none d-sm-inline">Mailpit</span>
			</RouterLink>
		</div>
		<div v-if="!errorMessage" class="col col-xl-5">
			<button class="btn btn-outline-light me-3 d-xl-none" title="Return to messages" @click="goBack()">
				<i class="bi bi-arrow-return-left"></i>
				<span class="ms-2 d-none d-lg-inline">Back</span>
			</button>
			<button class="btn btn-outline-light me-1 me-sm-2" title="Mark unread" @click="toggleRead()">
				<i class="bi bi-eye-slash me-md-2" :class="isRead ? 'bi-eye-slash' : 'bi-eye'"></i>
				<span class="d-none d-md-inline">Mark <template v-if="isRead">un</template>read</span>
			</button>
			<button
				v-if="mailbox.uiConfig.MessageRelay && mailbox.uiConfig.MessageRelay.Enabled"
				class="btn btn-outline-light me-1 me-sm-2"
				title="Release message"
				@click="initReleaseModal()"
			>
				<i class="bi bi-send me-md-2"></i>
				<span class="d-none d-md-inline">Release</span>
			</button>
			<button class="btn btn-outline-light me-1 me-sm-2" title="Delete message" @click="deleteMessage()">
				<i class="bi bi-trash-fill me-md-2"></i>
				<span class="d-none d-md-inline">Delete</span>
			</button>
		</div>
		<div v-if="!errorMessage" class="col-auto col-lg-4 col-xl-4 text-end">
			<div id="DownloadBtn" class="dropdown d-inline-block">
				<button
					type="button"
					class="btn btn-outline-light dropdown-toggle"
					data-bs-toggle="dropdown"
					aria-expanded="false"
				>
					<i class="bi bi-file-arrow-down-fill"></i>
					<span class="d-none d-md-inline ms-1">Download</span>
				</button>
				<ul class="dropdown-menu dropdown-menu-end">
					<li>
						<a
							:href="resolve('/api/v1/message/' + message.ID + '/raw?dl=1')"
							class="dropdown-item"
							title="Message source including headers, body and attachments"
						>
							Raw message
						</a>
					</li>
					<li v-if="message.HTML">
						<button class="dropdown-item" @click="downloadMessageBody(message.HTML, 'html')">
							HTML body
						</button>
					</li>
					<li v-if="message.HTML">
						<button class="dropdown-item" @click="screenshotMessageHTML()">HTML screenshot</button>
					</li>
					<li v-if="message.Text">
						<button class="dropdown-item" @click="downloadMessageBody(message.Text, 'txt')">
							Text body
						</button>
					</li>
					<template v-if="message.Attachments && message.Attachments.length">
						<li>
							<hr class="dropdown-divider" />
						</li>
						<li>
							<h6 class="dropdown-header">Attachments</h6>
						</li>
						<li v-for="part in message.Attachments" :key="part.PartID">
							<RouterLink
								:to="'/api/v1/message/' + message.ID + '/part/' + part.PartID"
								class="row m-0 dropdown-item d-flex"
								target="_blank"
								:title="part.FileName !== '' ? part.FileName : '[ unknown ]'"
								style="min-width: 350px"
							>
								<div class="col-auto p-0 pe-1">
									<i class="bi" :class="attachmentIcon(part)"></i>
								</div>
								<div class="col text-truncate p-0 pe-1">
									{{ part.FileName !== "" ? part.FileName : "[ unknown ]" }}
								</div>
								<div class="col-auto text-muted small p-0">
									{{ getFileSize(part.Size) }}
								</div>
							</RouterLink>
						</li>
					</template>
					<template v-if="message.Inline && message.Inline.length">
						<li>
							<hr class="dropdown-divider" />
						</li>
						<li>
							<h6 class="dropdown-header">Inline image<span v-if="message.Inline.length > 1">s</span></h6>
						</li>
						<li v-for="part in message.Inline" :key="part.PartID">
							<RouterLink
								:to="'/api/v1/message/' + message.ID + '/part/' + part.PartID"
								class="row m-0 dropdown-item d-flex"
								target="_blank"
								:title="part.FileName !== '' ? part.FileName : '[ unknown ]'"
								style="min-width: 350px"
							>
								<div class="col-auto p-0 pe-1">
									<i class="bi" :class="attachmentIcon(part)"></i>
								</div>
								<div class="col text-truncate p-0 pe-1">
									{{ part.FileName !== "" ? part.FileName : "[ unknown ]" }}
								</div>
								<div class="col-auto text-muted small p-0">
									{{ getFileSize(part.Size) }}
								</div>
							</RouterLink>
						</li>
					</template>
				</ul>
			</div>

			<RouterLink
				:to="'/view/' + previousID"
				class="btn btn-outline-light ms-1 ms-sm-2 me-1"
				:class="previousID ? '' : 'disabled'"
				title="View previous message"
			>
				<i class="bi bi-caret-left-fill"></i>
			</RouterLink>
			<RouterLink :to="'/view/' + nextID" class="btn btn-outline-light" :class="nextID ? '' : 'disabled'">
				<i class="bi bi-caret-right-fill" title="View next message"></i>
			</RouterLink>
		</div>
	</div>

	<div class="row flex-fill" style="min-height: 0">
		<div class="d-none d-xl-flex col-xl-3 h-100 flex-column">
			<div v-if="mailbox.uiConfig.Label" class="text-center badge text-bg-primary py-2 my-2 w-100">
				<div class="text-truncate fw-normal" style="line-height: 1rem">
					{{ mailbox.uiConfig.Label }}
				</div>
			</div>

			<div class="list-group my-2" :class="mailbox.uiConfig.Label ? 'mt-0' : ''">
				<button class="list-group-item list-group-item-action" @click="goBack()">
					<i class="bi bi-arrow-return-left me-1"></i>
					<span class="ms-1">
						Return to
						<template v-if="mailbox.searching">search</template>
						<template v-else>inbox</template>
					</span>
					<span
						v-if="mailbox.unread && !errorMessage"
						class="badge rounded-pill ms-1 float-end text-bg-secondary"
						title="Unread messages"
					>
						{{ formatNumber(mailbox.unread) }}
					</span>
				</button>
			</div>

			<div
				id="MessageList"
				ref="MessageList"
				class="flex-grow-1 overflow-y-auto px-1 me-n1"
				@scroll="scrollHandler"
			>
				<button v-if="liveLoaded >= 100" class="w-100 alert alert-warning small" @click="reloadWindow()">
					Reload to see newer messages
				</button>
				<template v-if="messagesList && messagesList.length">
					<div class="list-group">
						<RouterLink
							v-for="summary in messagesList"
							:id="summary.ID"
							:key="'summary_' + summary.ID"
							:to="'/view/' + summary.ID"
							class="row gx-1 message d-flex small list-group-item list-group-item-action message"
							:class="[summary.Read ? 'read' : '', isActive(summary.ID) ? 'active' : '']"
						>
							<div class="col overflow-x-hidden">
								<div class="text-truncate privacy small">
									<strong v-if="summary.From" :title="'From: ' + summary.From.Address">
										{{ summary.From.Name ? summary.From.Name : summary.From.Address }}
									</strong>
								</div>
							</div>
							<div class="col-auto small">
								<i v-if="summary.Attachments" class="bi bi-paperclip h6"></i>
								{{ getRelativeCreated(summary) }}
							</div>
							<div class="col-12 overflow-x-hidden">
								<div class="text-truncate privacy small">
									To: {{ getPrimaryEmailTo(summary) }}
									<span v-if="summary.To && summary.To.length > 1">
										[+{{ summary.To.length - 1 }}]
									</span>
								</div>
							</div>
							<div class="col-12 overflow-x-hidden mt-1">
								<div class="text-truncates small">
									<b>{{ summary.Subject !== "" ? summary.Subject : "[ no subject ]" }}</b>
								</div>
							</div>
							<div v-if="summary.Tags.length" class="col-12">
								<RouterLink
									v-for="t in summary.Tags"
									:key="t"
									class="badge me-1"
									:to="toTagUrl(t)"
									:style="
										mailbox.showTagColors
											? { backgroundColor: colorHash(t) }
											: { backgroundColor: '#6c757d' }
									"
									:title="'Filter messages tagged with ' + t"
									@click="pagination.start = 0"
								>
									{{ t }}
								</RouterLink>
							</div>
						</RouterLink>
					</div>
				</template>
			</div>

			<AboutMailpit />
		</div>

		<div class="col-xl-9 mh-100 ps-0 ps-md-2 pe-0">
			<div id="message-page" class="mh-100" style="overflow-y: auto">
				<template v-if="errorMessage">
					<h3 class="text-center my-3">
						{{ errorMessage }}
					</h3>
				</template>
				<Message v-else-if="message" :key="message.ID" :message="message" />
			</div>
		</div>
	</div>

	<AboutMailpit modals />
	<AjaxLoader :loading="loading" />
	<Release
		v-if="mailbox.uiConfig.MessageRelay && message"
		ref="ReleaseRef"
		:message="message"
		@delete="deleteMessage"
	/>
	<Screenshot v-if="message" ref="ScreenshotRef" :message="message" />
</template>
