package platform

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

// generatedFileType identifies a Magento generated file type by its filename suffix.
type generatedFileType int

const (
	genUnknown generatedFileType = iota
	genInterceptor
	genFactory
	genProxy
	genExtension
	genExtensionInterface
)

var (
	// Common patterns shared by all generated types.
	commonPatterns []*regexp.Regexp

	// Per-type patterns (in addition to common).
	typePatterns map[generatedFileType][]*regexp.Regexp
)

func init() {
	commonPatterns = compileAll([]string{
		`^\s*$`,                          // empty line
		`^\s*<\?php\s*$`,                 // <?php
		`^\s*namespace\s+`,               // namespace declaration
		`^\s*use\s+\\?[A-Z]`,            // use statement
		`^\s*(abstract\s+)?class\s+`,     // class declaration
		`^\s*interface\s+`,               // interface declaration
		`^\s*\{`,                          // opening brace
		`^\s*\}`,                          // closing brace
		`^\s*\*`,                          // PHPDoc comment line
		`^\s*/\*`,                         // block comment open
		`^\s*\*/`,                         // block comment close
		`^\s*//`,                          // line comment
		`^\s*#`,                           // hash comment
		`^\s*@`,                           // annotation
		`^\s*public\s+function\s+`,        // public method
		`^\s*protected\s+function\s+`,     // protected method
		`^\s*private\s+function\s+`,       // private method (rare but valid)
		`^\s*protected\s+\$`,              // protected property declaration
		`^\s*return\s+`,                   // return statement
		`^\s*return;`,                     // bare return
		`^\s*\)`,                          // closing paren (continuation)
		`^\s*\)\s*\{`,                     // closing paren + opening brace
		`^\s*\)\s*;`,                      // closing paren + semicolon
		`^\s*\?\s+\$this->`,              // ternary with $this->
		`^\s*:\s+\$this->`,               // ternary else with $this->
		`^\s*\$this->`,                    // $this-> property/method access
		`^\s*implements\s+`,               // implements (continuation)
		`^\s*extends\s+`,                  // extends (continuation)
	})

	interceptorPatterns := compileAll([]string{
		`\$this->pluginList->getNext\(`,
		`\$this->___callPlugins\(`,
		`\$this->___init\(\)`,
		`parent::`,
		`use\s+\\?Magento\\Framework\\Interception\\Interceptor`,
		`InterceptorInterface`,
		`func_get_args\(\)`,
	})

	factoryPatterns := compileAll([]string{
		`\$this->_objectManager`,
		`\$this->_instanceName`,
		`\$this->_objectManager->create\(`,
		`ObjectManagerInterface`,
	})

	proxyPatterns := compileAll([]string{
		`\$this->_getSubject\(\)->`,
		`\$this->_objectManager->get\(`,
		`\$this->_objectManager->create\(`,
		`\$this->_objectManager`,
		`ObjectManager::getInstance\(\)`,
		`\$this->_subject`,
		`\$this->_isShared`,
		`\$this->_instanceName`,
		`__sleep`,
		`__wakeup`,
		`__clone`,
		`__debugInfo`,
		`_resetState`,
		`_getSubject`,
		`NoninterceptableInterface`,
		`ObjectManagerInterface`,
	})

	extensionPatterns := compileAll([]string{
		`\$this->_get\(`,
		`\$this->setData\(`,
		`return\s+\$this;`,
		`AbstractSimpleObject`,
		`ExtensionInterface`,
	})

	extensionInterfacePatterns := compileAll([]string{
		`^\s*public\s+function\s+(get|set)\w+\(`,  // getter/setter declarations
		`ExtensionAttributesInterface`,
	})

	typePatterns = map[generatedFileType][]*regexp.Regexp{
		genInterceptor:        interceptorPatterns,
		genFactory:            factoryPatterns,
		genProxy:              proxyPatterns,
		genExtension:          extensionPatterns,
		genExtensionInterface: extensionInterfacePatterns,
	}
}

func compileAll(patterns []string) []*regexp.Regexp {
	rxs := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		rxs[i] = regexp.MustCompile(p)
	}
	return rxs
}

// classifyGenerated determines the generated file type from a relative path.
// Returns genUnknown if the path is not under generated/code/ or doesn't match
// a known suffix.
func classifyGenerated(relPath string) generatedFileType {
	if !strings.HasPrefix(relPath, "generated/code/") {
		return genUnknown
	}
	base := relPath[strings.LastIndex(relPath, "/")+1:]
	switch {
	case strings.HasSuffix(base, "ExtensionInterface.php"):
		return genExtensionInterface
	case strings.HasSuffix(base, "Extension.php"):
		return genExtension
	case strings.HasSuffix(base, "Interceptor.php"):
		return genInterceptor
	case strings.HasSuffix(base, "Factory.php"):
		return genFactory
	case strings.HasSuffix(base, "Proxy.php"):
		return genProxy
	default:
		return genUnknown
	}
}

// validateMagentoGenerated validates files under generated/code/ against
// known Magento code generation templates. Files that don't match known
// generated patterns are flagged.
func validateMagentoGenerated(relPath, absPath string, scanBuf []byte) (handled bool, hits []int, lines [][]byte) {
	ft := classifyGenerated(relPath)
	if ft == genUnknown {
		return false, nil, nil
	}

	f, err := os.Open(absPath)
	if err != nil {
		return true, nil, nil
	}
	defer f.Close()

	extra := typePatterns[ft]
	scanner := bufio.NewScanner(f)
	if len(scanBuf) > 0 {
		scanner.Buffer(scanBuf, len(scanBuf))
	}

	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Bytes()

		// If line matches any allowed pattern, it's template-generated — skip.
		if matchesAny(line, commonPatterns) || matchesAny(line, extra) {
			continue
		}

		// Unrecognized line — flag it.
		hits = append(hits, lineNo)
		lines = append(lines, append([]byte(nil), line...))
	}

	return true, hits, lines
}

func matchesAny(line []byte, patterns []*regexp.Regexp) bool {
	for _, rx := range patterns {
		if rx.Match(line) {
			return true
		}
	}
	return false
}
