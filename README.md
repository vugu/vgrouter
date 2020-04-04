# vgrouter

A URL router for Vugu.  As of April 2020 this is functional.  See test cases https://github.com/vugu/vgrouter/blob/master/rgen/rgen_test.go and https://github.com/vugu/vugu/tree/master/wasm-test-suite/test-012-router as examples. 

[![Travis CI](https://travis-ci.org/vugu/vgrouter.svg?branch=master)](https://travis-ci.org/vugu/vgrouter)

More documention will follow.

<!--
## NOTES:

Basic idea:
- RouteList deals with registering RouteHandlers and calling them.
- QueryBind deals with scanning structs for queries to be bound - synchronizes with a url.URL that it has on it
- Navigator interface has ability to nav to a URL (path and query) with options
- QueryUpdater interface has ability for component to say query bind stuff has changed and so push out to URL
- DefaultRouter has option of using path or fragment but otherwise merely ties the above together in one implementation.
- In wasm mode the RouteList stuff works the same, and the rest of it is just stubbed out nop or panic as appropriate.
- The idea is that applications create something like AppRouter which embeds vgrouter.DefaultRouter and add
  routes which then set fields on AppRouter.  Each "region" (e.g. header, footer, left navigation, main content, etc.) is a field on AppRouter, and then we need a mechanism to easily say "use this field, which has a value which implemented vugu.Builder, to render this component". Syntax probably something like: `<vg-component vg-expr="c.AppRouter.Footer"/>` (where Footer was set by one of the route handlers mentioned above).  Syntax still needs work, but I think the idea is sound.

Observations: If we treat a URL as just path and query (represented as string and url.Values), things become quite simple.
Navigate() updates both with the ability to pass options (needed for replacing history, e.g.), and the query stuff
(QueryUpdater interface and QueryBind) just deal with the query string and make it easy to do two-way binding.  This makes
some assumptions (e.g. we don't do anything with the fragment, and you can't automagically bind path params), but the other side is the common case of mapping a URL to show one or more components in various places, and binding params to indicate more detailed state - that common case is really easy.  And working without query binding is no more difficult than the Vue router.  The fact that Navigator is it's own interface also makes things a lot less coupled.

## More Notes:

- Can we do "automatic routing"? I.e. implied routes based on folder structure.
-->
