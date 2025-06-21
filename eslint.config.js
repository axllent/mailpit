import eslintConfigPrettier from "eslint-config-prettier/flat";
import neostandard, { resolveIgnoresFromGitignore } from "neostandard";
import vue from "eslint-plugin-vue";

export default [
	/* Baseline JS rules, provided by Neostandard */
	...neostandard({
		/* Allows references to browser APIs like `document` */
		env: ["browser"],

		/* We rely on .gitignore to avoid running against dist / dependency files */
		ignores: resolveIgnoresFromGitignore(),

		/* Disables a range of style-related rules, as we use Prettier for that */
		noStyle: true,

		/* Ensures we only lint JS and Vue files */
		files: ["**/*.js", "**/*.vue"],
	}),

	/* Vue-specific rules */
	...vue.configs["flat/recommended"],

	/* Prettier is responsible for formatting, so this disables any conflicting rules */
	eslintConfigPrettier,

	/* Our custom rules */
	{
		rules: {
			/* We prefer arrow functions for tidiness and consistency */
			"prefer-arrow-callback": "error",
		},
	},
];
