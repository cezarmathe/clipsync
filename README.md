# clipsync

Clipboard synchronization that JustWorks™ (for plain text).

**NOTE** This is a work-in-progress.

**NOTE** This is the quickest MVP that I could pull together, I needed a
clipboard synchronization solution that JustWorks™ ASAP.

**NOTE** Intended for plaintext clipboards.

**NOTE** Did a quick test with a Windows machine, this was treated as malware
and it also did not work. I'm not going to test further as I do not use Windows
myself but let me know if you find anything of interest on this!

## Usage

`go run main.go`

## How this works

clipsync advertises a `_clipsync._tcp` service on `local` via zeroconf to be
able to join other peers. It also runs a small HTTP server that updates the
clipboard when it receives a request. Each time the clipboard is updated on one
peer it sends an HTTP request to all other peers. Each time a peer receives an
HTTP request it updates the clipboard of the system.

## Why?

I use an M1 Mac as my main machine but I run an Arch Linux VM that serves as a
development box. I'm using Parallels as the hypervisor and clipboard
syncrhonization is not supported for Wayland - I'm running sway.

## License

[MIT](/LICENSE)
