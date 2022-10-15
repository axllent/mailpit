import axios from 'axios';
import { Modal } from 'bootstrap';


// FakeModal is used to return a fake Bootstrap modal
// if the ID returns nothing
function FakeModal() { }
FakeModal.prototype.hide = function () { alert('close fake modal') }
FakeModal.prototype.show = function () { alert('open fake modal') }

/* Common mixin functions used in apps */
const commonMixins = {
	data() {
		return {
			loading: 0
		}
	},

	methods: {
		getFileSize: function (bytes) {
			var i = Math.floor(Math.log(bytes) / Math.log(1024));
			return (bytes / Math.pow(1024, i)).toFixed(1) * 1 + ' ' + ['B', 'kB', 'MB', 'GB', 'TB'][i];
		},

		formatNumber: function (nr) {
			return new Intl.NumberFormat().format(nr);
		},

		// Ajax error message
		handleError: function (error) {
			// handle error
			if (error.response && error.response.data) {
				// The request was made and the server responded with a status code
				// that falls out of the range of 2xx
				if (error.response.data.Error) {
					alert(error.response.data.Error);
				} else {
					alert(error.response.data);
				}
			} else if (error.request) {
				// The request was made but no response was received
				// `error.request` is an instance of XMLHttpRequest in the browser and an instance of
				// http.ClientRequest in node.js
				alert('Error sending data to the server. Please try again.');
			} else {
				// Something happened in setting up the request that triggered an Error
				alert(error.message);
			}
		},

		// generic modal get/set function
		modal: function (id) {
			let e = document.getElementById(id);
			if (e) {
				return Modal.getOrCreateInstance(e);
			}
			// in case there are open/close actions
			return new FakeModal();
		},

		// generic modal get/set function
		offcanvas: function (id) {
			var e = document.getElementById(id);
			if (e) {
				return bootstrap.Offcanvas.getOrCreateInstance(e);
			}
			// in case there are open/close actions
			return new FakeModal();
		},

		/**
		 * Axios GET request
		 *
		 * @params string   url
		 * @params array    array parameters Object/array
		 * @params function callback function
		 */
		get: function (url, values, callback) {
			let self = this;
			self.loading++;
			axios.get(url, { params: values })
				.then(callback)
				.catch(self.handleError)
				.then(function () {
					// always executed
					if (self.loading > 0) {
						self.loading--;
					}
				});
		},

		/**
		 * Axios POST request
		 *
		 * @params string   url
		 * @params array    object/array values
		 * @params function callback function
		 */
		post: function (url, data, callback) {
			let self = this;
			self.loading++;
			axios.post(url, data)
				.then(callback)
				.catch(self.handleError)
				.then(function () {
					// always executed
					if (self.loading > 0) {
						self.loading--;
					}
				});
		},

		/**
		 * Axios DELETE request (REST only)
		 *
		 * @params string   url
		 * @params array    object/array values
		 * @params function callback function
		 */
		delete: function (url, data, callback) {
			let self = this;
			self.loading++;
			axios.delete(url, { data: data })
				.then(callback)
				.catch(self.handleError)
				.then(function () {
					// always executed
					if (self.loading > 0) {
						self.loading--;
					}
				});
		},

		/**
		 * Axios PUT request (REST only)
		 *
		 * @params string   url
		 * @params array    object/array values
		 * @params function callback function
		 */
		put: function (url, data, callback) {
			let self = this;
			self.loading++;
			axios.put(url, data)
				.then(callback)
				.catch(self.handleError)
				.then(function () {
					// always executed
					if (self.loading > 0) {
						self.loading--;
					}
				});
		},

		allAttachments: function (message) {
			let a = [];
			for (let i in message.Attachments) {
				a.push(message.Attachments[i]);
			}
			for (let i in message.OtherParts) {
				a.push(message.OtherParts[i]);
			}
			for (let i in message.Inline) {
				a.push(message.Inline[i]);
			}

			return a.length ? a : false;
		},

		isImage(a) {
			return a.ContentType.match(/^image\//);
		},

		attachmentIcon: function (a) {
			let ext = a.FileName.split('.').pop().toLowerCase();

			if (a.ContentType.match(/^image\//)) {
				return 'bi-file-image-fill';
			}
			if (a.ContentType.match(/\/pdf$/) || ext == 'pdf') {
				return 'bi-file-pdf-fill';
			}
			if (['doc', 'docx', 'odt', 'rtf'].includes(ext)) {
				return 'bi-file-word-fill';
			}
			if (['xls', 'xlsx', 'ods'].includes(ext)) {
				return 'bi-file-spreadsheet-fill';
			}
			if (['ppt', 'pptx', 'key', 'ppt', 'odp'].includes(ext)) {
				return 'bi-file-slides-fill';
			}
			if (['zip', 'tar', 'rar', 'bz2', 'gz', 'xz'].includes(ext)) {
				return 'bi-file-zip-fill';
			}
			if (a.ContentType.match(/^audio\//)) {
				return 'bi-file-music-fill';
			}
			if (a.ContentType.match(/^video\//)) {
				return 'bi-file-play-fill';
			}
			if (a.ContentType.match(/\/calendar$/)) {
				return 'bi-file-check-fill';
			}
			if (a.ContentType.match(/^text\//) || ['txt', 'sh', 'log'].includes(ext)) {
				return 'bi-file-text-fill';
			}

			return 'bi-file-arrow-down-fill';
		}
	}
}


export default commonMixins;
