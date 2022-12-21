# gopsx, a PlayStation 1 emulator written in Go

# Setup

1. Get a PlayStation 1 BIOS. I ended up using `SCPH1001.BIN`. The file should be exactly 512KB big. Checksums of `SCPH1001.BIN`:

    | MD5                                | SHA-1                                      |
    | ---------------------------------- | ------------------------------------------ |
    | `924e392ed05558ffdb115408c263dccf` | `10155d8d6e6e832d6ea66db9bc098321fb5e8ebf` |

    It should be fairly easy to find it on the web.

2. To boot the BIOS, run `<command> -bios "BIOS_PATH_HERE"`. The default BIOS path is `SCPH1001.BIN`, but i'll probably remove that sometime.

# Thanks

Special thanks to [simias](https://github.com/simias) for writing [this amazing guide](https://github.com/simias/psx-guide) to writing a PlayStation emulator, and to the [Nocash PSX spec](https://problemkaputt.de/psx.htm).