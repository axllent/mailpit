<script>
import { mailbox } from "../stores/mailbox.js";

export default {
	data() {
		return {
			favicon: false,
			iconPath: false,
			iconTextColor: "#ffffff",
			iconBgColor: "#dd0000",
			iconFontSize: 40,
			iconProcessing: false,
			iconTimeout: 500,
		};
	},

	computed: {
		count() {
			let i = mailbox.unread;
			if (i > 1000) {
				i = Math.floor(i / 1000) + "k";
			}

			return i;
		},
	},

	watch: {
		count() {
			if (!this.favicon || this.iconProcessing) {
				return;
			}

			this.iconProcessing = true;

			window.setTimeout(() => {
				this.icoUpdate();
			}, this.iconTimeout);
		},
	},

	mounted() {
		this.favicon = document.head.querySelector('link[rel="icon"]');
		if (this.favicon) {
			this.iconPath = this.favicon.href;
		}
	},

	methods: {
		async icoUpdate() {
			if (!this.favicon) {
				return;
			}

			if (!this.count) {
				this.iconProcessing = false;
				this.favicon.href = this.iconPath;
				return;
			}

			let fontSize = this.iconFontSize;
			// Draw badge text
			let textPaddingX = 7;
			const textPaddingY = 3;

			const strlen = this.count.toString().length;

			if (strlen > 2) {
				// if text >= 3 characters then reduce size and padding
				textPaddingX = 4;
				fontSize = strlen > 3 ? 30 : 36;
			}

			const canvas = document.createElement("canvas");
			canvas.width = 64;
			canvas.height = 64;

			const ctx = canvas.getContext("2d");

			// Draw base icon
			const icon = new Image();
			icon.src = this.iconPath;
			await icon.decode();

			ctx.drawImage(icon, 0, 0, 64, 64);

			// Measure text
			ctx.font = `${fontSize}px Arial, sans-serif`;
			ctx.textAlign = "right";
			ctx.textBaseline = "top";
			const textMetrics = ctx.measureText(this.count);

			// Draw badge
			const paddingX = 7;
			const paddingY = 4;
			const cornerRadius = 8;

			const width = textMetrics.width + paddingX * 2;
			const height = fontSize + paddingY * 2;
			const x = canvas.width - width;
			const y = canvas.height - height - 1;

			ctx.fillStyle = this.iconBgColor;
			ctx.roundRect(x, y, width, height, cornerRadius);
			ctx.fill();

			ctx.fillStyle = this.iconTextColor;
			ctx.fillText(this.count, canvas.width - textPaddingX, canvas.height - fontSize - textPaddingY);

			this.iconProcessing = false;

			this.favicon.href = canvas.toDataURL("image/png");
		},
	},
};
</script>
