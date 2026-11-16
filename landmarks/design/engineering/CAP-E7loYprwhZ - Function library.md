---
author: wasserbar
id: CAP-E7loYprwhZ
last_modified_by: wasserbar
links:
    refines:
        - CTX-jOPDBK9Zwu
name: Function library
owner: wasserbar
slug: function-library
status: approved
type: cap
---

## Description

Every function and field documented on Obsidian's Bases function
reference, organized by the type it operates on:

- **Global**: `escapeHTML(html)`, `date(date)`, `duration(value)`,
  `file(path)`, `html(html)`, `if(condition, trueResult, falseResult?)`,
  `image(path)`, `icon(name)`, `link(path, display?)`, `list(element)`,
  `max(...)`, `min(...)`, `now()`, `number(input)`, `today()`, `random()`
- **Any**: `.isTruthy()`, `.isType(type)`, `.toString()`
- **Date**: `.date()`, `.format(format)`, `.time()`, `.relative()`,
  `.isEmpty()`; fields `.year`/`.month`/`.day`/`.hour`/`.minute`/
  `.second`/`.millisecond`
- **String**: `.contains(v)`, `.containsAll(...)`, `.containsAny(...)`,
  `.endsWith(q)`, `.isEmpty()`, `.lower()`, `.replace(pattern, repl)`,
  `.repeat(n)`, `.reverse()`, `.slice(start, end?)`,
  `.split(sep, n?)`, `.startsWith(q)`, `.title()`, `.trim()`; field
  `.length`
- **Number**: `.abs()`, `.ceil()`, `.floor()`, `.isEmpty()`,
  `.round(digits)`, `.toFixed(precision)`
- **List**: `.contains(v)`, `.containsAll(...)`, `.containsAny(...)`,
  `.filter(expr)`, `.flat()`, `.isEmpty()`, `.join(sep)`, `.map(expr)`,
  `.reduce(expr, acc)`, `.reverse()`, `.slice(start, end?)`, `.sort()`,
  `.unique()`; field `.length`
- **Link**: `.asFile()`, `.linksTo(file)`
- **File**: `.asLink(display?)`, `.hasLink(otherFile)`,
  `.hasProperty(name)`, `.hasTag(...)`, `.inFolder(folder)`; fields
  `.name`, `.basename`, `.path`, `.folder`, `.ext`, `.size`,
  `.properties`, `.tags`, `.links`, `.ctime`, `.mtime`
- **Object**: `.isEmpty()`, `.keys()`, `.values()`
- **Regexp**: `.matches(value)`

Source: https://obsidian.md/help/bases/functions — re-verify against that
page before closing out this capability; it is the authoritative source,
not this doc.

## Actor

[[CAP-MB8sZOUUno]] — the evaluator calls into this library whenever a
filter/formula expression invokes a function.

## Rationale

Keeping the function set generic and spec-only (never cairn- or any other
single-consumer-aware — see [[DEC-RPUG8095er]]) is what keeps sill a
general-purpose tool rather than one project's dependency wearing a
standalone costume.

## Acceptance Criteria

- [ ] Every function in Obsidian's documented function reference is
      implemented with matching signature and behavior
- [ ] No function requires any schema knowledge beyond "this is a note with
      properties" — generic across any vault
- [ ] New functions are added only when documented by Obsidian, not
      speculatively