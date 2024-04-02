package main

import (
	"regexp"
)

var highlightPatterns = []string{
	`fopen\(`,
	`hex2bin\(`,
	`die\(`,
	`exec\(`,
	`chr\(`,
	`hexdec\(`,
	`eval\(`,
	`system\(`,
	`shell_exec\(`,
	`passthru\(`,
	`popen\(`,
	`proc_open\(`,
	`pcntl_exec\(`,
	`pcntl_fork\(`,
	`\$_[A-Z]`,
	`GLOBALS`,
	`\S"\s*\.\s*"\S`, // " . "
	`\S'\s*\.\s*'\S`, // ' . '
	`base64_`,
	`[a-zA-Z0-9\/\+\=]{40}`, // long base64? string
	// `@(unlink|include|mysql)`, already more generic one below
	// `../../..`, // too many fps
	// `curl_exec,
	`file_put_contents`,
	`file_get_contents`,
	`@[a-z_]{1,16}\(`, // suppressed function call
	`\$.\(\$.\(`,
	`call_user_func_array`,
	`\@\$\w{1,12}\(`, // suppressed dynamic function call
}

var highlightPatternsRe = compileRegexps(highlightPatterns)

func compileRegexps(patterns []string) []*regexp.Regexp {
	rxs := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		rxs[i] = regexp.MustCompile(p)
	}
	return rxs
}

func shouldHighlight(b []byte) bool {
	for _, rx := range highlightPatternsRe {
		if rx.Match(b) {
			return true
		}
	}
	return false
}
