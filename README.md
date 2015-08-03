This program is an integration between mutt and
[ash](https://github.com/seletskiy/ash).

Program will treat specified file as e-mail sent from Atlassian Stash about new
comment in a review and try to find URL to new comment.

If there is some URL found, then program will run ash with specially mocked
editor executable, so review will be scrolled down to the mentioned comment.

# Installation

## ArchLinux

Via [AUR](https://aur4.archlinux.org/packages/ash-mailcap) or self built
package:
```
git clone https://github.com/seletskiy/ash-mailcap
git checkout pkgbuld
makepkg
```

## go get

```
go get github.com/seletskiy/ash-mailcap
```

# Usage

Add following line to the ~/.mailcap file:

```
text/html; ash-mailcap \
               -x "i3-sensible-browser %s" %s;
```

**Note** the `i3-sensible-browser %s` string. It's the fallback action that
will be executed if passed html file doesn't contains any Stash links.

By default it's using vim. If you are not using vim, you need to write your own
wrapper (it's simple) and pass it using `-t` option. See `ash-mailcap -h` for
more details and examples.

## Autoview

You probably want to see context of the comments directly in the Stash
notification. Not a problem, just use
[ash-mailcap-autoview](https://github.com/seletskiy/ash-mailcap-autoview).

Install `ash-mailcap-autoview` and then add following to the ~/.mailcap:

```
text/html; ash-mailcap \
               -x "lynx --force-html --dump %s" \
               -t /usr/share/ash-mailcap-autoview/editor %s; \
               copiousoutput
```

Command `lynx --force-html --dump %s` will be used if it's not a Stash e-mail,  
`/usr/share/ash-mailcap-autoview/editor` comes with `ash-mailcap-autoview`
package.
