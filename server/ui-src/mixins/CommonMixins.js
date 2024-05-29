import axios from 'axios'
import dayjs from 'dayjs'
import ColorHash from 'color-hash'
import { Modal, Offcanvas } from 'bootstrap'
import {limitOptions} from "../stores/pagination";

// BootstrapElement is used to return a fake Bootstrap element
// if the ID returns nothing to prevent errors.
class BootstrapElement {
	constructor() { }
	hide() { }
	show() { }
}

// Set up the color hash generator lightness and hue to ensure darker colors
const colorHash = new ColorHash({ lightness: 0.3, saturation: [0.35, 0.5, 0.65] })

/* Common mixin functions used in apps */
export default {
	data() {
		return {
			loading: 0,
			tagColorCache: {},
		}
	},

	methods: {
		resolve: function (u) {
			return this.$router.resolve(u).href
		},

		searchURI: function (s) {
			return this.resolve('/search') + '?q=' + encodeURIComponent(s)
		},

		getFileSize: function (bytes) {
			if (bytes == 0) {
				return '0B'
			}
			var i = Math.floor(Math.log(bytes) / Math.log(1024))
			return (bytes / Math.pow(1024, i)).toFixed(1) * 1 + ' ' + ['B', 'kB', 'MB', 'GB', 'TB'][i]
		},

		formatNumber: function (nr) {
			return new Intl.NumberFormat().format(nr)
		},

		messageDate: function (d) {
			return dayjs(d).format('ddd, D MMM YYYY, h:mm a')
		},

		secondsToRelative: function (d) {
			return dayjs().subtract(d, 'seconds').fromNow()
		},

		tagEncodeURI: function (tag) {
			if (tag.match(/ /)) {
				tag = `"${tag}"`
			}

			return encodeURIComponent(`tag:${tag}`)
		},

		getSearch: function () {
			if (!window.location.search) {
				return false
			}

			const urlParams = new URLSearchParams(window.location.search)
			const q = urlParams.get('q')?.trim()
			if (!q) {
				return false
			}

			return q
		},

		getPaginationParams: function () {
			if (!window.location.search) {
				return null
			}

			const urlParams = new URLSearchParams(window.location.search)
			const start = parseInt(urlParams.get('start')?.trim(), 10)
			const limit = parseInt(urlParams.get('limit')?.trim(), 10)
			return {
				start: Number.isInteger(start) && start >= 0 ? start : null,
				limit: limitOptions.includes(limit) ? limit : null,
			}
		},

		// generic modal get/set function
		modal: function (id) {
			let e = document.getElementById(id)
			if (e) {
				return Modal.getOrCreateInstance(e)
			}
			// in case there are open/close actions
			return new BootstrapElement()
		},

		// close mobile navigation
		hideNav: function () {
			let e = document.getElementById('offcanvas')
			if (e) {
				Offcanvas.getOrCreateInstance(e).hide()
			}
		},

		/**
		 * Axios GET request
		 *
		 * @params string   url
		 * @params array    array parameters Object/array
		 * @params function callback function
		 * @params function error callback function
		 */
		get: function (url, values, callback, errorCallback) {
			let self = this
			self.loading++
			axios.get(url, { params: values })
				.then(callback)
				.catch(function (err) {
					if (typeof errorCallback == 'function') {
						return errorCallback(err)
					}

					self.handleError(err)
				})
				.then(function () {
					// always executed
					if (self.loading > 0) {
						self.loading--
					}
				})
		},

		/**
		 * Axios POST request
		 *
		 * @params string   url
		 * @params array    object/array values
		 * @params function callback function
		 */
		post: function (url, data, callback) {
			let self = this
			self.loading++
			axios.post(url, data)
				.then(callback)
				.catch(self.handleError)
				.then(function () {
					// always executed
					if (self.loading > 0) {
						self.loading--
					}
				})
		},

		/**
		 * Axios DELETE request (REST only)
		 *
		 * @params string   url
		 * @params array    object/array values
		 * @params function callback function
		 */
		delete: function (url, data, callback) {
			let self = this
			self.loading++
			axios.delete(url, { data: data })
				.then(callback)
				.catch(self.handleError)
				.then(function () {
					// always executed
					if (self.loading > 0) {
						self.loading--
					}
				})
		},

		/**
		 * Axios PUT request (REST only)
		 *
		 * @params string   url
		 * @params array    object/array values
		 * @params function callback function
		 */
		put: function (url, data, callback) {
			let self = this
			self.loading++
			axios.put(url, data)
				.then(callback)
				.catch(self.handleError)
				.then(function () {
					// always executed
					if (self.loading > 0) {
						self.loading--
					}
				})
		},

		// Ajax error message
		handleError: function (error) {
			// handle error
			if (error.response && error.response.data) {
				// The request was made and the server responded with a status code
				// that falls out of the range of 2xx
				if (error.response.data.Error) {
					alert(error.response.data.Error)
				} else {
					alert(error.response.data)
				}
			} else if (error.request) {
				// The request was made but no response was received
				alert('Error sending data to the server. Please try again.')
			} else {
				// Something happened in setting up the request that triggered an Error
				alert(error.message)
			}
		},

		allAttachments: function (message) {
			let a = []
			for (let i in message.Attachments) {
				a.push(message.Attachments[i])
			}
			for (let i in message.OtherParts) {
				a.push(message.OtherParts[i])
			}
			for (let i in message.Inline) {
				a.push(message.Inline[i])
			}

			return a.length ? a : false
		},

		isImage(a) {
			return a.ContentType.match(/^image\//)
		},

		attachmentIcon: function (a) {
			let ext = a.FileName.split('.').pop().toLowerCase()

			if (a.ContentType.match(/^image\//)) {
				return 'bi-file-image-fill'
			}
			if (a.ContentType.match(/\/pdf$/) || ext == 'pdf') {
				return 'bi-file-pdf-fill'
			}
			if (['doc', 'docx', 'odt', 'rtf'].includes(ext)) {
				return 'bi-file-word-fill'
			}
			if (['xls', 'xlsx', 'ods'].includes(ext)) {
				return 'bi-file-spreadsheet-fill'
			}
			if (['ppt', 'pptx', 'key', 'ppt', 'odp'].includes(ext)) {
				return 'bi-file-slides-fill'
			}
			if (['zip', 'tar', 'rar', 'bz2', 'gz', 'xz'].includes(ext)) {
				return 'bi-file-zip-fill'
			}
			if (['ics'].includes(ext)) {
				return 'bi-calendar-event'
			}
			if (a.ContentType.match(/^audio\//)) {
				return 'bi-file-music-fill'
			}
			if (a.ContentType.match(/^video\//)) {
				return 'bi-file-play-fill'
			}
			if (a.ContentType.match(/\/calendar$/)) {
				return 'bi-file-check-fill'
			}
			if (a.ContentType.match(/^text\//) || ['txt', 'sh', 'log'].includes(ext)) {
				return 'bi-file-text-fill'
			}

			return 'bi-file-arrow-down-fill'
		},

		// Returns a hex color based on a string.
		// Values are stored in an array for faster lookup / processing.
		colorHash: function (s) {
			if (this.tagColorCache[s] != undefined) {
				return this.tagColorCache[s]
			}
			this.tagColorCache[s] = colorHash.hex(s)

			return this.tagColorCache[s]
		},
	}
}
