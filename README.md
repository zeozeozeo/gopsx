# gopsx, a PlayStation 1 emulator written in Go

# Setup

1. Get a PlayStation 1 BIOS. I ended up using `SCPH1001.BIN`. The file should be exactly 512KB big. Checksums of `SCPH1001.BIN`:

    | MD5                                | SHA-1                                      |
    | ---------------------------------- | ------------------------------------------ |
    | `924e392ed05558ffdb115408c263dccf` | `10155d8d6e6e832d6ea66db9bc098321fb5e8ebf` |

    It should be fairly easy to find it on the web.

2. To boot the BIOS, run `<command> -bios "BIOS_PATH_HERE"`. The default BIOS path is `SCPH1001.BIN`, but i'll probably remove that sometime.

# Status

CPU, GPU, DMA, timers, CD-ROM, controllers and interrupts are partially implemented. The boot logo is being rendered correctly, but it doesn't have any textures yet.

![Boot animation](https://cdn.discordapp.com/attachments/783966433641365504/1056857431906455622/image.png)

# Other

Default keyboard keymappings:

|  Gamepad  |    Keyboard     |
| :-------: | :-------------: |
|   Start   |    Backspace    |
|  Select   |   Right Shift   |
|  DPadUp   |    Arrow Up     |
| DPadRight |   Arrow Right   |
| DPadDown  |   Arrow Down    |
| DPadLeft  |   Arrow Left    |
|    L2     |  Keypad Divide  |
|    R2     | Keypad Multiply |
|    L1     |    Keypad 7     |
|    R1     |    Keypad 9     |
| Triangle  |    Keypad 8     |
|  Circle   |    Keypad 6     |
|   Cross   |    Keypad 2     |
|  Square   |    Keypad 4     |

You can change them in the `main.go` file, but it would be great to be able to do that from the CLI

# Thanks

Special thanks to [simias](https://github.com/simias) for writing [this amazing guide](https://github.com/simias/psx-guide) to writing a PlayStation emulator, and to the [Nocash PSX spec](https://problemkaputt.de/psx.htm).
