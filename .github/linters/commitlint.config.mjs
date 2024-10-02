export default {
	extends: ['@commitlint/config-conventional'],
	helpUrl: 'https://www.conventionalcommits.org/',
	ignores: [
		(msg) => /Signed-off-by: dependabot\[bot]/m.test(msg),
	],
	rules: {
		'header-max-length': [2, 'always', 72],
		'body-max-line-length': [2, 'always', 80],
		'subject-case': [2, 'never', ['start-case', 'pascal-case', 'upper-case']],
		'trailer-exists': [2, 'always', 'Signed-off-by:'],
	},
}
