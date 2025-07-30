import eslintConfigPrettier from "eslint-config-prettier/flat";
import globals from "globals";
import { includeIgnoreFile } from "@eslint/compat";
import js from "@eslint/js";
import vue from "eslint-plugin-vue";
import { fileURLToPath } from "node:url";

const gitignorePath = fileURLToPath(new URL(".gitignore", import.meta.url));

export default [
	/* Use .gitignore to prevent linting of irrelevant files */
	includeIgnoreFile(gitignorePath, ".gitignore"),

	/* ESLint's recommended rules */
	{
		files: ["**/*.js", "**/*.vue"],
		languageOptions: { globals: { ...globals.browser, ...globals.node } },
		rules: js.configs.recommended.rules,
	},

	/* Vue-specific rules */
	...vue.configs["flat/recommended"],

	/* Prettier is responsible for formatting, so we disable conflicting rules */
	eslintConfigPrettier,

	/* Our custom rules */
	{
		rules: {
			/* Always use arrow functions for tidiness and consistency */
			"prefer-arrow-callback": "error",

			/* Always use camelCase for variable names */
			camelcase: [
				"error",
				{
					ignoreDestructuring: false,
					ignoreGlobals: true,
					ignoreImports: false,
					properties: "never",
				},
			],

			/* The default case in switch statements must always be last */
			"default-case-last": "error",

			/* Always use dot notation where possible (e.g. `obj.val` over `obj['val']`) */
			"dot-notation": "error",

			/* Always use `===` and `!==` for comparisons unless unambiguous */
			eqeqeq: ["error", "smart"],

			/* Never use `eval()` as it violates our CSP and can lead to security issues */
			"no-eval": "error",
			"no-implied-eval": "error",

			/* Prevents accidental use of template literals in plain strings, e.g. "my ${var}" */
			"no-template-curly-in-string": "error",

			/* Avoid unnecessary ternary operators */
			"no-unneeded-ternary": "error",

			/* Avoid unused expressions that have no purpose */
			"no-unused-expressions": "error",

			/* Always use `const` or `let` to make scope behaviour clear */
			"no-var": "error",

			/* Always use shorthand syntax for objects where possible, e.g. { a, b() { } } */
			"object-shorthand": "error",

			/* Always use `const` for variables that are never reassigned */
			"prefer-const": "error",
		},
	},
];
