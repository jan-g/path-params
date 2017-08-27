# Efficient cachable path parameters

Top-line summary: indexed DB possible (no table scans), partial trie data-structure in memory loaded
on demand, linear costs for parsing.

## Introduction

This is a demonstration of the caching and parsing of route parameters in an efficient fashion.
Additionally, a single interpretation of path parameters is supplied (ie, there is no ambiguity
in the parsing rules).

The implementation is split into two pieces: a database layer which just requires keyed lookups
from some prefix (ie, no scans); and a cache layer which holds a partial tree in memory.

The assumption here is that route alterations happen much less frequently than route lookups.

## Costs of updates on the database

A database update for a path with _n_ elements (ie, `/p1/p2/p3/.../pn`) will require O(_n_) insertions
into a single indexed table; this corresponds to an entry for each of the path prefixes, `/p1`, `/p1/p2`,
and so on.

Deletions work similarly.

Every top-level app is considered to have a monotonically increasing generation; a new update will bump
the generation on every prefix on the way to the route.

The current 'database' implementation is an in-memory one only for purposes of illustration.

## Costs of lookups at the cache layer

The cache layer constructs a partial view of the route tree; specifically, it caches nodes on the
path to any route leaf it's asked to look up (assuming that route exists).

If the cache currently has an up-to-date path for a particular route, the query can be answered directly
from memory in a structure traversal that's linear in the length of the route.

It is possible that some part of the route prefix may be out-of-date; the cache structure will detect
this and perform a number of database lookups corresponding to the length of the route prefix which is
out-of-date. Nothing else is thrown away; all other route information can be kept.

### Example

Suppose we install the route `/a/b/c/d/e/f/g`. We make a query for that route; the cache structure
will be populated from the database. Cache nodes will be of generation 1. The cache structure at this
time looks like this:

    /a(1)/b(1)/c(1)/d(1)/e(1)/f(1)/g(1) -> { route data here }

Now suppose a second route is installed, `/a/b/X/Y/Z`. In the *database*, the following prefixes will have
these generations:

    /a                  2
    /a/b                2
    /a/b/c              1
    /a/b/c/d            1
    /a/b/c/d/e          1
    /a/b/c/d/e/f        1
    /a/b/c/d/e/f/g      1
    /a/b/X              2
    /a/b/X/Y            2
    /a/b/X/Y/Z          2

Suppose a second query is made for the initial route; and suppose further that the data for the top-level
node is stale (see below for cache-invalidation strategies). Then the cache layer will issue two queries,
for `/a` and `/a/b`, at which point it will discover that the generation of the remainder of its subtree
is still valid. The cache structure will look like this:

    /a(2)/b(2)/c(1)/ ...etc... /g(1) -> { route data here }

A request for `/a/b/X/Y/Z` will follow the tree as far as the `b` node, then issue requests for `X`, `Y` and `Z`.

In short: the number of database trips is limited to the number of nodes that are out-of-date in the cache
for the current path, or that it currently does not hold. (In practice we'd expect this to be zero under the
assumptions above.)


## Cache busting strategies

### Time-to-live

This is implemented in the current sketch; top-level (app) nodes have a timeout. If a request comes
in for that app, the database will only be contacted in two situations: either if the cache currently
has no entry (positive or negative) for that app; or if the current entry has passed its time-to-live.
All other nodes can be updated deterministically - there's no need for further timeout complications.

### Cache-busting notifications from the database layer

An alternative strategy would be for the database to publish cache invalidation notices on a pub-sub
channel which all caches subscribe to; in this situation, top-level nodes may be marked as out-of-date
which will trigger a refresh on the next query for a route belonging to that app. Absent those notifications,
the cache entries could live forever.

### Unconditional fetches of the top-level nodes

This approach is equivalent to setting the positive TTL to 0; it touches the database (once, under normal
operation with stable routes) per lookup. The rest of the datastructure will be held in cache where possible.
All results are "live".

### Cache pruning

The current implementation throws away nodes pruned from the prefix tree when it discovers them; it
keeps an entry for each app it is asked about (even if that entry is an expired negative one).

There is no reason why this scheme couldn't be augmented with a cache whose entries are more volatile;
the resulting cost would be more trips to the database layer, but that may be considered a suitable
compromise. The current implementation keeps everything it loads until it determines that it's a
subtree that has been pruned.


## Path parameter semantics

This implementation supports a couple of path variable types. There's the single-path-element one,
encoded as `:`; additionally, there's a _remainder of the input_ variable, encoded as `&`.

When hunting for a path, the most specific match is taken at each step. Fixed path entries are considered
to be more specific than single-element `:` path variables, which are in turn more specific than the `&`
rest-of-path.

In practice, this is unlikely to be a burden. Consider the following route set:

    1. /graph
    2. /graph/view
    3. /graph/:/stage/:
    4. /graph/&             ERROR: cannot match anything
    5. /graph/:/&

Then the following requests would correspond to these paths:

    /graph                      matches 1
    /graph/view                 matches 2
    /graph/view/                unmatched
    /graph/view/foo             unmatched
    /graph/2934/stage/4372      matches 3
    /graph/4234                 unmatched (but adding a route of /graph/: would match this)
    /graph/4234/                matches 5 (rest parameter is "")
    /graph/4234/x/y/z           matches 5 (rest parameter is "x/y/z")

Tweaks to the semantics of trailing slashes are of course possible; the current approach favours
brevity of implementation. 