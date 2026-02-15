# Magento Platform Notes

## Generated Code (`generated/`)

Magento 2 auto-generates PHP classes into `generated/code/` via `bin/magento setup:di:compile`. These are prime hiding spots for malware because they're typically ignored during code review.

### File Types

| Type | Count (sample) | Suffix |
|---|---|---|
| Interceptors | 3369 | `*/Interceptor.php` |
| Factories | 1429 | `*Factory.php` |
| Extensions | 193 | `*Extension.php` |
| ExtensionInterfaces | 191 | `*ExtensionInterface.php` |
| Proxies | 180 | `*/Proxy.php` |

There are also metadata files under `generated/metadata/` (plugin-list, area configs) which are serialized PHP arrays.

Only files matching these suffixes should exist under `generated/code/`. Any other filename is immediately suspicious.

### Content Patterns

Every line in every generated file matches one of a small set of rigid patterns. Analysis of 5381 files (116K lines) showed 83% repetition and zero lines outside the expected templates.

**Interceptors** extend the original class, implement `InterceptorInterface`, and wrap public methods with plugin chain dispatch:
```
$pluginInfo = $this->pluginList->getNext($this->subjectType, 'methodName');
return $pluginInfo ? $this->___callPlugins('methodName', func_get_args(), $pluginInfo) : parent::methodName(...);
```

**Factories** are ~15-line templates with only the class name and `$instanceName` default varying. Body is always `return $this->_objectManager->create($this->_instanceName, $data);`.

**Proxies** have a fixed skeleton (constructor, `__sleep`, `__wakeup`, `__clone`, `__debugInfo`, `_getSubject`) plus delegating methods that always follow `$this->_getSubject()->methodName(args)` or `return $this->_getSubject()->methodName(args)`.

**Extensions/ExtensionInterfaces** are nearly empty — just a class/interface declaration extending a framework base type.

### Validation Strategy

Because the structure is completely rigid, generated code can be validated through pattern matching without a PHP parser:

1. **Filename validation**: only expected suffixes should exist under `generated/code/`
2. **Line-level validation**: every non-comment, non-whitespace line must match one of ~10 regex patterns per file type
3. **Negative pattern matching**: flag lines containing `eval(`, `base64_decode(`, `gzinflate(`, `$_GET`, `$_POST`, `$_REQUEST`, `shell_exec(`, `system(`, `exec(`, `file_put_contents(` etc.

The variable parts (class names, method signatures, constructor parameters) are legitimate PHP identifiers and type references — they follow predictable syntax even if the exact names are installation-specific.

### What We Don't Need

- No PHP parser required
- No reimplementation of Magento's DI compiler
- No need to know which modules are installed
- No need to pre-compute expected generated files per version

The generated code is so formulaic that anomaly detection (flagging anything that doesn't fit the template) is more effective than whitelist matching.
