<script>
import { Toast } from 'bootstrap';

export default {
	props: {
		message: Object
	},

	mounted() {
		let self = this;
		let el = document.getElementById('messageToast');
		if (el) {
			el.addEventListener('hidden.bs.toast', () => {
				self.$emit("clearMessageToast");
			})

			let b = Toast.getOrCreateInstance(el);
			b.show();
		}
	}
}
</script>

<template>
	<div class="toast-container position-fixed bottom-0 end-0 p-3">
		<div id="messageToast" class="toast" role="alert" aria-live="assertive" aria-atomic="true">
			<div class="toast-header">
				<i class="bi bi-envelope-exclamation-fill me-2"></i>
				<strong class="me-auto"><a :href="'#' + message.ID">New message</a></strong>
				<small class="text-body-secondary">now</small>
				<button type="button" class="btn-close" data-bs-dismiss="toast" aria-label="Close"></button>
			</div>

			<div class="toast-body">
				<div>
					<a :href="'#' + message.ID" class="d-block text-truncate text-muted">
						<template v-if="message.Subject != ''">{{ message.Subject }}</template>
						<template v-else>[ no subject ]</template>
					</a>
				</div>
			</div>
		</div>
	</div>
</template>
