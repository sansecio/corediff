package main

import (
	"bytes"
	"regexp"
)

var (
	highlightPatternsRe = compileRegexps([]string{
		// php
		`\$_[A-Z]`,       // $_GET, $_POST, etc.
		`\S"\s*\.\s*"\S`, // " . "
		`\S'\s*\.\s*'\S`, // ' . '
		`@$?\w{1,16}\(`,  // suppressed function call
		`\$.\(\$.\(`,     // $x($y(
		`\@\$\w{1,12}\(`, // suppressed dynamic function call
		`\/\*\s*\w+\s*\*\/.+\/\*\s*\w+\s*\*\/[^\s]+`,                       // comment obfuscation
		`include\s{1,10}["'\x60](\w|\/)+\.(png|jpeg|svg|jpg|webp)["'\x60]`, // include php as image

		// common
		`[a-zA-Z0-9\/\+\=]{25,}`, // long base64 string
		`(\\x[A-Z0-9]{2}){15,}`,  // long hex string
		`(_0x\w{4,8}.+){4,}`,     // multiple obfuscated variables
	})
	highlightPatternsLit = [][]byte{
		// php
		[]byte(`system(`),
		[]byte(`fopen(`),
		[]byte(`hex2bin(`),
		[]byte(`die(`),
		[]byte(`chr(`),
		[]byte(`hexdec(`),

		[]byte(`exec`),
		[]byte(`shell_exec`),
		[]byte(`passthru`),
		[]byte(`popen`),
		[]byte(`proc_open`),
		[]byte(`pcntl_exec`),
		[]byte(`escapeshellcmd`),
		[]byte(`preg_replace`),
		[]byte(`create_function`),
		[]byte(`call_user_func_array`),

		[]byte(`base64_`),
		[]byte(`strrev`),
		[]byte(`str_rot13`),
		[]byte(`htmlspecialchars_decode`),

		[]byte(`file_get_contents`),
		[]byte(`file_put_contents`),
		[]byte(`fwrite`),
		[]byte(`fread`),
		[]byte(`fgetc`),
		[]byte(`fgets`),
		[]byte(`fscanf`),
		[]byte(`fgetss`),
		[]byte(`fpassthru`),
		[]byte(`readfile`),

		[]byte(`gzuncompress`),
		[]byte(`gzinflate`),
		[]byte(`gzdecode`),
		[]byte(`readgzfile`),
		[]byte(`gzwrite`),
		[]byte(`gzfile`),

		[]byte(`umask(0)`),
		[]byte(`chmod($`),
		[]byte(`chown($`),
		[]byte(`chgrp($`),
		[]byte(`unlink(`),
		[]byte(`rmdir(`),
		[]byte(`mkdir(`),
		[]byte(`stream_get_meta_data`),

		[]byte(`GLOBALS`),

		[]byte(`$obirninja`),
		[]byte(`$pass`),
		[]byte(`<?php @'$`),

		// js
		[]byte(`atob`),
		[]byte(`btoa`),
		[]byte(`String.fromCharCode(`),
		[]byte(`jQuery.getScript(`),

		// common
		[]byte(`../../../../../../`),
		[]byte(`eval`),
	}
)

func compileRegexps(patterns []string) []*regexp.Regexp {
	rxs := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		rxs[i] = regexp.MustCompile(p)
	}
	return rxs
}

func shouldHighlight(b []byte) bool {
	for _, p := range highlightPatternsLit {
		if bytes.Contains(b, p) {
			return true
		}
	}
	for _, rx := range highlightPatternsRe {
		if rx.Match(b) {
			return true
		}
	}

	return false
}
