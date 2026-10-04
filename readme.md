# Temporally Ambiguous CMS

This is a CMS designed to be one piece of the puzzle in bringing old web
culture into the modern era. It is designed to be what I call "medium effort
publishing," and sits somewhere between [Wordpress](https://wordpress.com/) and
[Hugo](https://gohugo.io/). This project is designed to be secure by default,
easy to deploy for simple use cases, and has the ability to be deployed as a
12-factor app, supporting high availability through replication.

For more information about what I aim to accomplish with this software, see my
first blog post hosted by this software here:

[https://shaunkeys.com/posts/hello-world/](https://shaunkeys.com/posts/hello-world/)

## Under Construction

To-do list:

* [x] Markdown parsing
* [x] Push markdown articles from CLI
  * [x] Declared asset resolution
  * [ ] Implicit asset resolution (from references in article content)
  * [x] Asset upload
  * [ ] Article management (post-upload)
* [x] Asset storage
  * [x] Local file asset storage
  * [x] S3 asset storage
* [x] Web article server
  * [x] Asset serving
  * [x] RSS/Atom/JSON feeds
  * [ ] Article listing (partially implemented)
* [ ] Downloadables & archival
* [ ] TUI browser
  * [ ] SSH
  * [ ] Telnet
  * [ ] Local CLI (using connect-rpc to fetch raw markdown for alternative rendering)
  * [ ] Xterm.js (WebAssembly build of local CLI possibly? Otherwise just Telnet over WebSockets)
* [ ] ActivityPub (to be expanded on later)

Please also note that this project is undergoing heavy development at the
moment, so I will not be creating a migration for every database change.
Smooth upgrade paths are only guaranteed across tagged versions. You have been
warned.

## Design Goals

For the short-term, I'm hard-coding a lot of things like templates and such,
because I want this working as soon as possible for myself. However, as time
goes on, I will be making it more configurable. Users of the software will be
able to add their own templates and customize the look of the site in any way
they wish.

For my personal website, I'm going for a modern take on retro web design. It is
designed to be fast and responsive, with little to no JavaScript requirement so
it can be viewed in almost any software the user wishes. I have confirmed it
is perfectly readable on the web, in `lynx`, and [FeedFlow](https://www.feedflow.dev/).

## Terminal Browsing

I would also like to have a TUI for browsing articles. This will be accessible
via SSH and Telnet, as well as a custom CLI. The reason for the custom CLI is
that with Markdown, I think we can do a lot better than Lynx (not because lynx
is bad; it solves a very different problem). [charm.sh](https://charm.sh/) has
their [Glamour library](https://github.com/charmbracelet/glamour), which can be
used to display markdown beautifully in the terminal. This is *much* easier to
implement well than trying to display arbitrary HTML. Now, so far everything
I've brought up can be done with SSH or Telnet. However, there are three more
problems: IPv4 exhaustion, lazy IPv6 adoption, and hyperlinks.

This is where I think the CLI comes in. The CLI can use ConnectRPC, which means
it can communicate over a reverse proxy like Cloudflare (and by extension,
Cloudflare Tunnel). This will enable people on a residential connection to
solve several problems:

* Lack of a public IPv4 address
  * We've been out of IPv4 addresses for a while now, and many ISPs are running
  out of their reserves as well. This means many people don't actually have a
  publicly-routable IP address that they can accept connections on. Cloudflare
  tunnel gets around this by having *you* connect to *them*, giving them a path
  back into *your* network.
* Dynamic IP addressing
  * Most residential connections have dynamic IP addresses. This means that
  your address may change at any time, rendering your DNS configuration
  useless. This can also be remedied by Cloudflare tunnel, but dynamic DNS
  software/services are an option too if you don't trust Cloudflare.
* DDoS attacks
  * Cloudflare can mask your actual IP addresses, and will take the brunt of
  the impact in the event of a DDoS attack on your site.

A custom CLI will also allow hyperlinking, and reduce latency and load on the
server and network, since only the source material will be transferred instead
of full screen renders on every keypress.

## More to come

This project is nowhere near finished, so expect to see more over time.
